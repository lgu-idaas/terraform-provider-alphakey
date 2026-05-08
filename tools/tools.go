//go:build tools

package tools

import (
	// Testing dependencies - kept here to ensure they remain in go.mod
	_ "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	_ "pgregory.net/rapid"
)
