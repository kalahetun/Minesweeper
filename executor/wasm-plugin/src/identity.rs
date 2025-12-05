//! Envoy Identity Module
//!
//! Extracts service identity from Envoy node metadata for policy filtering.
//! Used to implement service-level policy targeting in multi-pod deployments.

use proxy_wasm::hostcalls::get_property;

/// Represents the identity of an Envoy sidecar instance.
/// Extracted from Envoy node metadata at plugin startup.
#[derive(Debug, Clone, Default)]
pub struct EnvoyIdentity {
    /// Kubernetes workload name (e.g., "frontend")
    pub workload_name: String,
    /// Kubernetes namespace (e.g., "demo")
    pub namespace: String,
    /// Pod name (for logging, optional)
    pub pod_name: Option<String>,
    /// Istio cluster ID (e.g., "frontend.demo")
    pub cluster: Option<String>,
}

impl EnvoyIdentity {
    /// Creates a new EnvoyIdentity by extracting metadata from Envoy node properties.
    ///
    /// Uses proxy-wasm `get_property` to access:
    /// - `node.metadata.WORKLOAD_NAME` → workload_name
    /// - `node.metadata.NAMESPACE` → namespace
    /// - `node.metadata.NAME` → pod_name (optional)
    /// - `node.cluster` → cluster (optional)
    ///
    /// If metadata is unavailable, defaults to "*" (wildcard) for required fields.
    pub fn from_envoy_metadata() -> Self {
        let workload_name = get_property(vec!["node", "metadata", "WORKLOAD_NAME"])
            .ok()
            .flatten()
            .and_then(|v| String::from_utf8(v).ok())
            .unwrap_or_else(|| "*".to_string());

        let namespace = get_property(vec!["node", "metadata", "NAMESPACE"])
            .ok()
            .flatten()
            .and_then(|v| String::from_utf8(v).ok())
            .unwrap_or_else(|| "*".to_string());

        let pod_name = get_property(vec!["node", "metadata", "NAME"])
            .ok()
            .flatten()
            .and_then(|v| String::from_utf8(v).ok());

        let cluster = get_property(vec!["node", "cluster"])
            .ok()
            .flatten()
            .and_then(|v| String::from_utf8(v).ok());

        let identity = Self {
            workload_name,
            namespace,
            pod_name,
            cluster,
        };

        log::info!(
            "EnvoyIdentity extracted: workload={}, namespace={}, pod={:?}, cluster={:?}",
            identity.workload_name,
            identity.namespace,
            identity.pod_name,
            identity.cluster
        );

        identity
    }

    /// Checks if this identity matches a given ServiceSelector.
    ///
    /// Matching logic:
    /// - "*" or empty string in selector matches any value
    /// - Exact string match required otherwise
    ///
    /// Returns true if both service and namespace match.
    pub fn matches_selector(&self, selector: &ServiceSelector) -> bool {
        let service_matches = selector.service.is_empty()
            || selector.service == "*"
            || selector.service == self.workload_name;

        let namespace_matches = selector.namespace.is_empty()
            || selector.namespace == "*"
            || selector.namespace == self.namespace;

        service_matches && namespace_matches
    }

    /// Returns a display string for logging purposes.
    pub fn display(&self) -> String {
        format!(
            "{}.{}",
            self.workload_name,
            self.namespace
        )
    }
}

/// Service selector for policy targeting.
/// Specifies which services a policy applies to.
#[derive(Debug, Clone, Default, serde::Deserialize, serde::Serialize)]
pub struct ServiceSelector {
    /// Service/workload name to match. "*" or empty matches all.
    #[serde(default)]
    pub service: String,
    /// Namespace to match. "*" or empty matches all.
    #[serde(default)]
    pub namespace: String,
}

impl ServiceSelector {
    /// Creates a new wildcard selector that matches all services.
    pub fn wildcard() -> Self {
        Self {
            service: "*".to_string(),
            namespace: "*".to_string(),
        }
    }

    /// Creates a selector for a specific service in a specific namespace.
    pub fn new(service: impl Into<String>, namespace: impl Into<String>) -> Self {
        Self {
            service: service.into(),
            namespace: namespace.into(),
        }
    }

    /// Returns true if this selector matches all services (wildcard).
    pub fn is_wildcard(&self) -> bool {
        (self.service.is_empty() || self.service == "*")
            && (self.namespace.is_empty() || self.namespace == "*")
    }

    /// Normalizes empty values to "*" for consistent matching.
    pub fn normalize(&mut self) {
        if self.service.is_empty() {
            self.service = "*".to_string();
        }
        if self.namespace.is_empty() {
            self.namespace = "*".to_string();
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_wildcard_selector_matches_all() {
        let identity = EnvoyIdentity {
            workload_name: "frontend".to_string(),
            namespace: "demo".to_string(),
            pod_name: None,
            cluster: None,
        };

        let selector = ServiceSelector::wildcard();
        assert!(identity.matches_selector(&selector));
    }

    #[test]
    fn test_exact_match() {
        let identity = EnvoyIdentity {
            workload_name: "frontend".to_string(),
            namespace: "demo".to_string(),
            pod_name: None,
            cluster: None,
        };

        let selector = ServiceSelector::new("frontend", "demo");
        assert!(identity.matches_selector(&selector));
    }

    #[test]
    fn test_service_mismatch() {
        let identity = EnvoyIdentity {
            workload_name: "frontend".to_string(),
            namespace: "demo".to_string(),
            pod_name: None,
            cluster: None,
        };

        let selector = ServiceSelector::new("backend", "demo");
        assert!(!identity.matches_selector(&selector));
    }

    #[test]
    fn test_namespace_mismatch() {
        let identity = EnvoyIdentity {
            workload_name: "frontend".to_string(),
            namespace: "demo".to_string(),
            pod_name: None,
            cluster: None,
        };

        let selector = ServiceSelector::new("frontend", "production");
        assert!(!identity.matches_selector(&selector));
    }

    #[test]
    fn test_partial_wildcard() {
        let identity = EnvoyIdentity {
            workload_name: "frontend".to_string(),
            namespace: "demo".to_string(),
            pod_name: None,
            cluster: None,
        };

        // Service-only selector (namespace wildcard)
        let selector = ServiceSelector::new("frontend", "*");
        assert!(identity.matches_selector(&selector));

        // Namespace-only selector (service wildcard)
        let selector = ServiceSelector::new("*", "demo");
        assert!(identity.matches_selector(&selector));
    }

    #[test]
    fn test_empty_values_treated_as_wildcard() {
        let identity = EnvoyIdentity {
            workload_name: "frontend".to_string(),
            namespace: "demo".to_string(),
            pod_name: None,
            cluster: None,
        };

        let selector = ServiceSelector {
            service: "".to_string(),
            namespace: "".to_string(),
        };
        assert!(identity.matches_selector(&selector));
    }

    #[test]
    fn test_normalize() {
        let mut selector = ServiceSelector {
            service: "".to_string(),
            namespace: "".to_string(),
        };
        selector.normalize();
        assert_eq!(selector.service, "*");
        assert_eq!(selector.namespace, "*");
    }
}
