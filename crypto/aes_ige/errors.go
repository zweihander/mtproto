package ige

import "fmt"

var (
	ErrDataTooSmall     = fmt.Errorf("AES256IGE: data too small")
	ErrDataNotDivisible = fmt.Errorf("AES256IGE: data not divisible by block size")
)
