// utils.go
package postgresStores

import "os"

// GetEnv retrieves an environment variable or returns a default value if not found
func GetEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}