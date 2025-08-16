// This binary is for testing the config parsing functionality

use std::fs;

// Include our modules directly
mod config {
    include!("../config.rs");
}

fn main() {
    // Read the test configuration
    let config_data = fs::read("../test-config.json")
        .expect("Failed to read test-config.json from parent directory");
    
    // Try to parse it using our config module
    match config::CompiledRuleSet::from_slice(&config_data) {
        Ok(ruleset) => {
            println!("✅ Successfully parsed configuration!");
            println!("Version: {}", ruleset.version);
            println!("Number of rules: {}", ruleset.rules.len());
            
            for (i, rule) in ruleset.rules.iter().enumerate() {
                println!("\n📋 Rule {}: {}", i + 1, rule.name);
                
                // Check path matching
                if let Some(ref path) = rule.match_condition.path {
                    if let Some(ref prefix) = path.prefix {
                        println!("  Path prefix: {}", prefix);
                    }
                    if let Some(ref exact) = path.exact {
                        println!("  Path exact: {}", exact);
                    }
                    if let Some(ref regex_str) = path.regex {
                        println!("  Path regex: {}", regex_str);
                        if path.compiled_regex.is_some() {
                            println!("  ✅ Regex compiled successfully");
                        } else {
                            println!("  ❌ Regex compilation failed");
                        }
                    }
                }
                
                // Check method matching
                if let Some(ref method) = rule.match_condition.method {
                    if let Some(ref exact) = method.exact {
                        println!("  Method: {}", exact);
                    }
                }
                
                // Check header matching
                if let Some(ref headers) = rule.match_condition.headers {
                    for header in headers {
                        println!("  Header {}: {:?}", header.name, header.exact);
                    }
                }
                
                // Check fault configuration
                println!("  Fault percentage: {}%", rule.fault.percentage);
                
                if let Some(ref abort) = rule.fault.abort {
                    println!("  Abort: {} - {:?}", abort.http_status, abort.body);
                }
                
                if let Some(ref delay) = rule.fault.delay {
                    println!("  Delay: {}", delay.fixed_delay);
                    if let Some(duration_ms) = delay.parsed_duration_ms {
                        println!("  ✅ Parsed duration: {}ms", duration_ms);
                    } else {
                        println!("  ❌ Duration parsing failed");
                    }
                }
            }
        }
        Err(e) => {
            println!("❌ Failed to parse configuration: {}", e);
            std::process::exit(1);
        }
    }
}
