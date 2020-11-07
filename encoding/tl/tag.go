package tl

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	tagName          = "tl"
	encodedInBitflag = "encoded_in_bitflag"
)

type tagInfo struct {
	index            int
	encodedInBitflag bool
	required         bool
	ignore           bool
}

func parseFlagTag(s string) (info tagInfo, err error) {
	vals := strings.Split(s, ",")
	if len(vals) == 0 {
		err = fmt.Errorf("bad tl_flag: %s", s)
		return
	}

	if vals[0] == "-" {
		info.ignore = true
		return
	}

	// flag index check
	if strings.HasPrefix(vals[0], "flag:") {
		trimmed := vals[0][5:]
		info.index, err = strconv.Atoi(trimmed)
		if err != nil {
			err = fmt.Errorf("invalid flag index '%s': %w", trimmed, err)
			return
		}

		if len(vals) == 2 {
			if vals[1] == encodedInBitflag {
				info.encodedInBitflag = true
			} else {
				err = fmt.Errorf("parse flag second option: expected '%s': got '%s'", encodedInBitflag, vals[1])
				return
			}
		}
	} else {
		err = fmt.Errorf("invalid tag: %s", s)
	}

	return
}
