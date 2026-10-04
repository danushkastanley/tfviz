package aws

import (
	"regexp"
	"testing"
)

// Stems are file names in the official pack, without size or extension.
var stemPattern = regexp.MustCompile(`^(Arch|Res)_[A-Za-z0-9-]+(_[A-Za-z0-9-]+)?$`)

func TestIconStemsCoverOnlySupportedTypes(t *testing.T) {
	stems := IconStems()
	for typ, list := range stems {
		if _, ok := Lookup(typ); !ok {
			t.Errorf("%s has icons but no adapter", typ)
		}
		if len(list) == 0 {
			t.Errorf("%s has an empty icon list", typ)
		}
		for _, stem := range list {
			if !stemPattern.MatchString(stem) {
				t.Errorf("%s: %q does not look like an AWS pack file name", typ, stem)
			}
		}
	}
	// Security groups and their rules have no official icon; everything
	// else supported should be recognisable.
	for _, typ := range Types() {
		if _, ok := stems[typ]; !ok && !noOfficialIcon[typ] {
			t.Errorf("%s has no icon mapping", typ)
		}
	}
}

var noOfficialIcon = map[string]bool{
	"aws_security_group": true, "aws_security_group_rule": true,
	"aws_vpc_security_group_ingress_rule": true, "aws_vpc_security_group_egress_rule": true,
}

func TestIconStemsReturnsACopy(t *testing.T) {
	IconStems()["aws_vpc"] = nil
	if len(IconStems()["aws_vpc"]) == 0 {
		t.Fatal("callers can change the shared table")
	}
}
