package helpers

import "os"

//GetAuthTenant returns auth tenant
func GetAuthTenant() string {
	return os.Getenv("AUTH_TENANT")
}
