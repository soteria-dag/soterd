// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"reflect"
	"testing"
)

func TestNodeId(t *testing.T) {
	var node = newNode("A")

	if node.GetId() != "A" {
		t.Errorf("GetId returned wrong node id. Expecting %s, got %s", "A", node.GetId())
	}
}

func TestNodeGetChildren(t *testing.T) {
	var parent = newNode("Parent")
	var c1 = newNode("c1")
	var c2 = newNode("c2")

	parent.children[c1] = keyExists
	parent.children[c2] = keyExists

	var children = parent.getChildren().elements()
	var expected = []*Node{c1, c2}
	if !reflect.DeepEqual(children, expected) {
		t.Errorf("Returned wrong child nodes. Expecting %v, got %v",
			GetIds(expected), GetIds(children))
	}
}