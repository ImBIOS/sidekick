// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// axNode is one uiautomator <node>. Only the fields agents act on survive.
type axNode struct {
	Class     string   `xml:"class,attr"`
	Text      string   `xml:"text,attr"`
	Desc      string   `xml:"content-desc,attr"`
	Bounds    string   `xml:"bounds,attr"`
	Clickable string   `xml:"clickable,attr"`
	Nodes     []axNode `xml:"node"`
}

// outNode is the agent-facing element: stable @e refs in document order,
// hierarchy preserved so agents see stacking.
type outNode struct {
	Ref       string    `json:"ref"`
	Class     string    `json:"class"`
	Text      string    `json:"text,omitempty"`
	Desc      string    `json:"desc,omitempty"`
	Bounds    string    `json:"bounds,omitempty"`
	Clickable bool      `json:"clickable,omitempty"`
	Children  []outNode `json:"children,omitempty"`
}

// uiaToJSON converts `uiautomator dump` XML into the AX tree agents read.
// Refs (@e1, @e2, ...) follow document order and are stable per dump;
// agents must re-read after every screen change, same contract as iOS.
func uiaToJSON(xmlDoc []byte) ([]byte, error) {
	// uiautomator prepends nothing, but tolerates a leading dump path line.
	if i := bytes.Index(xmlDoc, []byte("<?xml")); i > 0 {
		xmlDoc = xmlDoc[i:]
	}
	var root struct {
		Nodes []axNode `xml:"node"`
	}
	dec := xml.NewDecoder(bytes.NewReader(xmlDoc))
	dec.Strict = false
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("uiautomator xml: %w", err)
	}
	counter := 0
	var walk func(x axNode) outNode
	walk = func(x axNode) outNode {
		counter++
		o := outNode{
			Ref:    "@e" + strconv.Itoa(counter),
			Class:  shortClass(x.Class),
			Text:   x.Text,
			Desc:   x.Desc,
			Bounds: x.Bounds,
		}
		if x.Clickable == "true" {
			o.Clickable = true
		}
		for _, c := range x.Nodes {
			o.Children = append(o.Children, walk(c))
		}
		return o
	}
	var children []outNode
	for _, n := range root.Nodes {
		children = append(children, walk(n))
	}
	if children == nil {
		children = []outNode{}
	}
	return json.Marshal(map[string]any{"nodes": children, "refs": counter})
}

// shortClass trims android.widget.Button to Button.
func shortClass(c string) string {
	if i := strings.LastIndex(c, "."); i >= 0 {
		return c[i+1:]
	}
	return c
}
