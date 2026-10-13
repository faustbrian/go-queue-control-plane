package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Exercise the actual operator document loader before any database acquisition.
func TestR30RetentionSecondsAdmissionBeforeDurationConversion(t *testing.T) {
	minimum := int64(minimumRetention / time.Second)
	maximum := int64(maximumRetention / time.Second)
	for _, seconds := range []int64{math.MinInt64, -(1 << 55) + minimum, -1, 0, minimum - 1, minimum, minimum + 1, maximum, maximum + 1, math.MaxInt64 / int64(time.Second), math.MaxInt64/int64(time.Second) + 1, (1 << 55) + minimum, math.MaxInt64} {
		t.Run(strconv.FormatInt(seconds, 10), func(t *testing.T) {
			document := fmt.Sprintf(`{"tenants":[{"id":"tenant","retention_seconds":%d,"batch_size":1,"max_batches":1,"legal_hold":false}]}`, seconds)
			policies, err := loadRetentionPolicies(strings.NewReader(document), 4096)
			if seconds < minimum || seconds > maximum {
				if !errors.Is(err, ErrInvalidRetentionDocument) || len(policies) != 0 {
					t.Fatalf("out-of-range seconds admitted: seconds=%d policies=%v error=%v", seconds, policies, err)
				}
				return
			}
			if err != nil || len(policies) != 1 || policies[0].RetainFor != time.Duration(seconds)*time.Second {
				t.Fatalf("valid retention boundary changed: %v %v", policies, err)
			}
		})
	}
}

func FuzzR30RetentionSecondsAdmission(f *testing.F) {
	for _, seconds := range []int64{3600, 315360000, -1, -(1 << 55) + 3600, (1 << 55) + 3600, math.MinInt64, math.MaxInt64} {
		f.Add(seconds)
	}
	f.Fuzz(func(t *testing.T, seconds int64) {
		document := fmt.Sprintf(`{"tenants":[{"id":"tenant","retention_seconds":%d,"batch_size":1,"max_batches":1,"legal_hold":false}]}`, seconds)
		policies, err := loadRetentionPolicies(strings.NewReader(document), 4096)
		admitted := seconds >= int64(minimumRetention/time.Second) && seconds <= int64(maximumRetention/time.Second)
		if !admitted {
			if !errors.Is(err, ErrInvalidRetentionDocument) || len(policies) != 0 {
				t.Fatal("out-of-range seconds returned a retention policy")
			}
		} else if err != nil || len(policies) != 1 || policies[0].RetainFor != time.Duration(seconds)*time.Second {
			t.Fatal("valid scalar did not preserve exact policy duration")
		}
	})
}
