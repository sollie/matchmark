package service

import "github.com/sollie/matchmark/internal/names"

func NormalizeLastName(s string) string { return names.LastName(s) }

func NormalizeFirstName(s string) string { return names.FirstName(s) }

func validShooterNames(first, last string) bool {
	return first == NormalizeFirstName(first) && last == NormalizeLastName(last)
}
