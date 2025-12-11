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
    /// Whether identity was successfully extracted (not using fallback wildcards)
    pub is_valid: bool,
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
        log::debug!("Extracting Envoy identity from node metadata...");

        // Try to extract workload name
        let workload_result = get_property(vec!["node", "metadata", "WORKLOAD_NAME"]);
        let workload_name = workload_result
            .ok()
            .flatten()
            .and_then(|v| String::from_utf8(v).ok());

        let has_workload = workload_name.is_some();
        if !has_workload {
            log::warn!("Failed to extract WORKLOAD_NAME from Envoy metadata, using wildcard");
        }

        // Try to extract namespace
        let namespace_result = get_property(vec!["node", "metadata", "NAMESPACE"]);
        let namespace = namespace_result
            .ok()
            .flatten()
            .and_then(|v| String::from_utf8(v).ok());

        let has_namespace = namespace.is_some();
        if !has_namespace {
            log::warn!("Failed to extract NAMESPACE from Envoy metadata, using wildcard");
        }

        // Try to extract pod name (optional, for debugging)
        let pod_name = get_property(vec!["node", "metadata", "NAME"])
            .ok()
            .flatten()
            .and_then(|v| String::from_utf8(v).ok());

        // Try to extract cluster ID (optional)
        let cluster = get_property(vec!["node", "cluster"])
            .ok()
            .flatten()
            .and_then(|v| String::from_utf8(v).ok());

        let is_valid = has_workload && has_namespace;

        let identity = Self {
            workload_name: workload_name.unwrap_or_else(|| "*".to_string()),
            namespace: namespace.unwrap_or_else(|| "*".to_string()),
            pod_name,
            cluster,
            is_valid,
        };

        // Log at appropriate level based on success/failure
        if is_valid {
            log::info!(
                "EnvoyIdentity successfully extracted: workload={}, namespace={}, pod={:?}, cluster={:?}",
                identity.workload_name,
                identity.namespace,
                identity.pod_name,
                identity.cluster
            );
        } else {
            log::warn!(
                "EnvoyIdentity extraction incomplete (fail-open mode): workload={}, namespace={}, pod={:?}. Only wildcard policies will apply.",
                identity.workload_name,
                identity.namespace,
                identity.pod_name
            );
        }

        identity
    }

    /// Returns true if identity was successfully extracted from Envoy metadata.
    /// If false, only wildcard policies should be applied (fail-open mode).
    pub fn is_valid(&self) -> bool {
        self.is_valid
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

        let result = service_matches && namespace_matches;

        log::debug!(
            "Selector match check: identity={}.{} vs selector={}.{} => {}",
            self.workload_name,
            self.namespace,
            selector.service,
            selector.namespace,
            if result { "MATCH" } else { "NO MATCH" }
        );

        result
    }

    /// Returns a display string for logging purposes.
    pub fn display(&self) -> String {
        format!("{}.{}", self.workload_name, self.namespace)
    }

    /// Returns detailed display string including pod name and validity.
    pub fn display_detailed(&self) -> String {
        format!(
            "{}.{} (pod: {}, valid: {})",
            self.workload_name,
            self.namespace,
            self.pod_name.as_deref().unwrap_or("unknown"),
            self.is_valid
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
            is_valid: true,
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
            is_valid: true,
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
            is_valid: true,
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
            is_valid: true,
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
            is_valid: true,
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
            is_valid: true,
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
