package utils

import "strings"

func GenerateName(RBACRuleName, BN, Kind, RN string) string {
	return strings.Join([]string{RBACRuleName, BN, Kind, RN}, "-")
}
