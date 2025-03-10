package routines

import (
	"encoding/xml"
	"errors"
	"fmt"
)

type (
	// Candidate represents routine manifest to be included into the execution plan
	Candidate struct {
		Label    string            `json:"-"`
		Summary  string            `json:"summary"`
		Targets  []string          `json:"targets"`
		Metadata map[string]string `json:"metadata"`
	}
)

type (
	XMLRule struct {
		Node  xml.Name `xml:"rule"`
		Class string   `xml:"class,attr"`
		Name  string   `xml:"name,attr"`
		// StrAttrs should contain following attributes:
		//  * summary
		//  * name, i.e. target name
		StrAttrs []*XMLRuleStringAttr `xml:"string"`

		// ListAttrs contains label_list attribute
		// Note: currently only targets list is expected as the only label_list attribute
		ListAttrs []*XMLRuleLabelListAttr `xml:"list"`
	}

	XMLRuleStringAttr struct {
		Node  xml.Name `xml:"string"`
		Name  string   `xml:"name,attr"`
		Value string   `xml:"value,attr"`
	}

	XMLRuleLabelAttr struct {
		Node  xml.Name `xml:"label"`
		Value string   `xml:"value,attr"`
	}

	XMLRuleLabelListAttr struct {
		Node   xml.Name            `xml:"list"`
		Name   string              `xml:"name,attr"`
		Labels []*XMLRuleLabelAttr `xml:"label"`
	}
)

func (r XMLRule) toCandidate() (c Candidate, err error) {
	for _, attr := range r.ListAttrs {
		switch attr.Name {
		case "targets":
			for _, l := range attr.Labels {
				c.Targets = append(c.Targets, l.Value)
			}
		}
	}

	if len(c.Targets) == 0 {
		return c, errors.New("no targets provided")
	}

	for _, attr := range r.StrAttrs {
		switch attr.Name {
		case "summary":
			c.Summary = attr.Value
		}
	}

	c.Label = r.Name

	return c, nil
}

// DecodeCandidatesXML uses bazel query's xml output to get all the raw attributes of the routine manifest rule
// without analyzing and building all the dependency graph
//
// Example of the xml output:
// -------
// <?xml version="1.1" encoding="UTF-8" standalone="no"?>
// <query version="2">
//
//	<rule class="routine_manifest" location="/test/b/BUILD.bazel:32:17" name="//test/b:release-staging">
//	    <string name="name" value="release-staging"/>
//	    <string name="summary" value="service-B"/>
//	    <list name="targets">
//	        <label value="//test/b:prepare-deployment"/>
//	        <label value="//test/b:ensure-deployment"/>
//	    </list>
//	    <rule-input name="//test/b:ensure-deployment"/>
//	    <rule-input name="//test/b:prepare-deployment"/>
//	</rule>
//
// </query>
func DecodeCandidatesXML(rules []*XMLRule) ([]Candidate, error) {
	var candidates []Candidate
	for _, r := range rules {
		if r.Class != ManifestRuleKind {
			continue
		}

		c, err := r.toCandidate()
		if err != nil {
			return nil, fmt.Errorf("failed to decode candidate(%s): %w", r.Name, err)
		}

		candidates = append(candidates, c)
	}
	return candidates, nil
}
