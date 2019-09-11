// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"bytes"
	"encoding/gob"
	"sort"
)

// NODESET
type nodeSet struct {
	nodes map[*Node]struct{}
}

func newNodeSet() *nodeSet {
	return &nodeSet{
		nodes: make(map[*Node]struct{}),
	}
}

func (nset *nodeSet) size() int {
	return len(nset.nodes)
}

func (nset *nodeSet) add(node *Node) {
	if node == nil {
		return
	}

	if nset.contains(node) {
		return
	}

	nset.nodes[node] = keyExists
	//fmt.Printf("Added node %s\n", node.id)
}

func (nset *nodeSet) remove(node *Node) {
	// Remove this node
	delete(nset.nodes, node)
}

// returns elements of set, sorted by id
func (nset *nodeSet) elements() []*Node {
	nodes := make([]*Node, len(nset.nodes))
	index := 0
	for k := range nset.nodes {
		nodes[index] = k
		index += 1
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].GetId() < nodes[j].GetId() {
			return true
		} else {
			return false
		}
	})

	return nodes
}

func (nset *nodeSet) contains(node *Node) bool {
	_, ok := nset.nodes[node]
	return ok
}

// returns nset - nset2
func (nset *nodeSet) difference(nset2 *nodeSet) *nodeSet {
	diff := newNodeSet()
	if nset2 == nil {
		return diff
	}

	for k := range nset.nodes {
		if !nset2.contains(k) {
			diff.add(k)
		}
	}
	return diff
}

// returns nset intersection nset2
func (nset *nodeSet) intersection(nset2 *nodeSet) *nodeSet {
	intersection := newNodeSet()
	if nset2 == nil {
		return intersection
	}

	for k := range nset.nodes {
		if nset2.contains(k) {
			intersection.add(k)
		}
	}
	return intersection

}

func (nset *nodeSet) clone() *nodeSet {
	clone := newNodeSet()
	for k := range nset.nodes {
		clone.add(k)
	}

	return clone
}

// ORDEREDNODESET
type orderedNodeSet struct {
	nodes map[*Node]int
	order map[int]*Node
	counter int
}

func newOrderedNodeSet() *orderedNodeSet {
	return &orderedNodeSet {
		nodes: make(map[*Node]int),
		order: make(map[int]*Node),
		counter: 0,
	}
}

// add the node to the ordered node set, if it's not null and doesn't already exist
func (ons *orderedNodeSet) add(node *Node) {
	if node == nil {
		return
	}

	if ons.contains(node) {
		return
	}

	ons.counter += 1
	ons.nodes[node] = ons.counter
	ons.order[ons.counter] = node
}

// clone returns a copy of the ordered node set
func (ons *orderedNodeSet) clone() *orderedNodeSet {
	clone := newOrderedNodeSet()
	for _, k := range ons.elements() {
		clone.add(k)
	}

	return clone
}

// contains returns true if the node is in the ordered node set
func (ons *orderedNodeSet) contains(node *Node) bool {
	_, ok := ons.nodes[node]
	return ok
}

// difference returns ons - ons2
func (ons *orderedNodeSet) difference(ons2 *orderedNodeSet) *orderedNodeSet {
	diff := newOrderedNodeSet()
	if ons2 == nil {
		return ons.clone()
	}

	for _, k := range ons.elements() {
		if !ons2.contains(k){
			diff.add(k)
		}
	}

	return diff
}

// elements returns a list of nodes in the order they were added to the ordered node set
func (ons *orderedNodeSet) elements() []*Node {
	var indexes = make([]int, 0, len(ons.order))
	var nodes = make([]*Node, 0, len(ons.order))
	for k := range ons.order {
		indexes = append(indexes, k)
	}

	sort.Ints(indexes)

	for _, v := range indexes {
		nodes = append(nodes, ons.order[v])
	}
	return nodes
}

// intersection returns ons intersection ons2
func (ons *orderedNodeSet) intersection(ons2 *orderedNodeSet) *orderedNodeSet {
	intersection := newOrderedNodeSet()
	if ons2 == nil {
		return intersection
	}

	for _, k := range ons.elements() {
		if ons2.contains(k) {
			intersection.add(k)
		}
	}

	return intersection
}

// remove a node from the ordered node set
func (ons *orderedNodeSet) remove(node *Node) {
	if node == nil {
		return
	}

	// Remove this node
	i, ok := ons.nodes[node]
	if ok {
		delete(ons.order, i)
	}
	delete(ons.nodes, node)
}

// size returns the size of the ordered node set
func (ons *orderedNodeSet) size() int {
	return len(ons.nodes)
}

// An alternative to using a container/list List type, for managing a list of nodes
type nodeList struct {
	elements []*Node
}

func newNodeList(cap int) *nodeList {
	return &nodeList{
		elements: make([]*Node, 0, cap),
	}
}

// at returns the node at the given index, or nil if there's no Node there.
func (nl *nodeList) at(i int) *Node {
	if i > len(nl.elements) - 1 || i < 0 {
		return nil
	}

	return nl.elements[i]
}

// delete removes a node at the given index, and returns it
func (nl *nodeList) delete(i int) *Node {
	end := len(nl.elements) - 1
	n := nl.elements[i]
	// Copy values from the deletion point to the left by one
	copy(nl.elements[i:], nl.elements[i+1:])
	// Dereference the last value
	nl.elements[end] = nil
	// Truncate the slice
	nl.elements = nl.elements[:end]

	return n
}

// insert adds a Node at the given index.
// This method of insertion avoids creating a new slice, so garbage collection isn't involved.
func (nl *nodeList) insert(i int, n *Node) {
	// Add a nil value to the end of the slice, to make room for the new Node.
	nl.elements = append(nl.elements, nil)
	// Copy values from the insertion point to the right by one
	copy(nl.elements[i+1:], nl.elements[i:])
	// Set the value at the insertion point
	nl.elements[i] = n
}

// push adds a Node to the end of the list
func (nl *nodeList) push(n *Node) {
	nl.elements = append(nl.elements, n)
}

// pop removes a Node from the end of the list, and returns it. nil is returned if there's no Node to pop.
func (nl *nodeList) pop() *Node {
	size := len(nl.elements)
	if size == 0 {
		return nil
	}

	// This method of deletion is used instead of calling nl.Delete(), because it's faster.
	end := size - 1
	n := nl.elements[end]
	nl.elements[end] = nil
	nl.elements = nl.elements[0:end]

	return n
}

// shift removes a Node from the front of the list, and returns it
func (nl *nodeList) shift() *Node {
	if len(nl.elements) == 0 {
		return nil
	}

	// This method of deletion is used instead of calling nl.Delete(), because it's faster.
	n := nl.elements[0]
	nl.elements[0] = nil
	nl.elements = nl.elements[1:]
	return n
}

// size returns the size of the nodeList
func (nl *nodeList) size() int {
	return len(nl.elements)
}

// unshift adds a Node to the front of the list
func (nl *nodeList) unshift(n *Node) {
	nl.insert(0, n)
}

type StringSet struct {
	members map[string]struct{}
}

// NewStringSet returns a new StringSet
func NewStringSet() *StringSet {
	return &StringSet{
		members: make(map[string]struct{}),
	}
}

// This decodes a slice of bytes into a new StringSet instance.
//
// This is meant to help load a persisted blueCache entry from disk,
// and is the opposite of the ss.Encode() method.
func DecodeStringSet(blob []byte) (*StringSet, error) {
	ss := NewStringSet()
	buf := bytes.NewBuffer(blob)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&ss.members)
	if err != nil {
		return nil, err
	}

	return ss, nil
}

// Add adds an element to the set
func (ss *StringSet) Add(aString string) {
	if ss.Contains(aString) {
		return
	}

	ss.members[aString] = keyExists
}

// Remove removes an element from the set
func (ss *StringSet) Remove(aString string) {
	delete(ss.members, aString)
}

// Size returns the number of elements in the set
func (ss *StringSet) Size() int {
	return len(ss.members)
}

// Contains returns true if the element is in the set
func (ss *StringSet) Contains(aString string) bool {
	_, ok := ss.members[aString]
	return ok
}

// Elements returns elements of set, sorted alphabetically
func (ss *StringSet) Elements() []string {
	size := len(ss.members)
	members := make([]string, size, size)
	index := 0
	for k := range ss.members {
		members[index] = k
		index += 1
	}

	sort.Strings(members)

	return members
}

// Difference returns set - other
func (ss *StringSet) Difference(other *StringSet) *StringSet {
	diff := NewStringSet()
	if other == nil {
		for k := range ss.members {
			diff.Add(k)
		}
		return diff
	}

	for k := range ss.members {
		if !other.Contains(k) {
			diff.Add(k)
		}
	}
	return diff
}

// Intersection returns set intersection other
func (ss *StringSet) Intersection(other *StringSet) *StringSet {
	intersection := NewStringSet()
	if other == nil {
		return intersection
	}

	for k := range ss.members {
		if other.Contains(k) {
			intersection.Add(k)
		}
	}
	return intersection
}

// Encode returns bytes in gob format representing the set members.
// This is meant to help persist blueSet cache entries to disk.
//
// We specify an Encode() method instead of running gob.NewEncoder().Encode()
// on the StringSet directly, because gob only encodes exported fields
// and we'd prefer to keep the set internals unexported.
//
// For the reverse operation, use DecodeStringSet function to create a new StringSet from bytes.
func (ss *StringSet) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(ss.members)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type OrderedStringSet struct {
	members map[string]int
	order map[int]string
	counter int
}

// NewOrderedStringSet returns a new empty ordered string set
func NewOrderedStringSet(members ...string) *OrderedStringSet {
	oss := OrderedStringSet{
		members: make(map[string]int),
		order: make(map[int]string),
		counter: 0,
	}

	for _, m := range members {
		oss.Add(m)
	}

	return &oss
}

// Add adds a member to the set if doesn't already exist
func (oss *OrderedStringSet) Add(aString string) {
	if oss.Contains(aString) {
		return
	}

	oss.counter += 1
	oss.members[aString] = oss.counter
	oss.order[oss.counter] = aString
}

// Clone returns a copy of the set
func (oss *OrderedStringSet) Clone() *OrderedStringSet {
	clone := NewOrderedStringSet()
	for _, k := range oss.Elements() {
		clone.Add(k)
	}

	return clone
}

// Contains returns true if the member is in the set
func (oss *OrderedStringSet) Contains(aString string) bool {
	_, ok := oss.members[aString]
	return ok
}

// Difference returns a set with the members of this set that aren't in setB
func (oss *OrderedStringSet) Difference(setB *OrderedStringSet) *OrderedStringSet {
	diff := NewOrderedStringSet()
	if setB == nil {
		return oss.Clone()
	}

	for _, m := range oss.Elements() {
		if !setB.Contains(m) {
			diff.Add(m)
		}
	}

	return diff
}

// Elements returns members of the set in the order they were added
func (oss *OrderedStringSet) Elements() []string {
	size := len(oss.order)
	var indices = make([]int, size, size)
	var order = make([]string, size, size)
	count := 0
	for index := range oss.order {
		indices[count] = index
		count += 1
	}

	sort.Ints(indices)

	count = 0
	for _, index := range indices {
		order[count] = oss.order[index]
		count += 1
	}

	return order
}

// Intersection returns a set with the members that exist in both this set and setB
func (oss *OrderedStringSet) Intersection(setB *OrderedStringSet) *OrderedStringSet {
	intersection := NewOrderedStringSet()
	if setB == nil {
		return intersection
	}

	for _, m := range oss.Elements() {
		if setB.Contains(m) {
			intersection.Add(m)
		}
	}

	return intersection
}

// Remove removes a member from the set
func (oss *OrderedStringSet) Remove(aString string) {
	i, ok := oss.members[aString]
	if ok {
		delete(oss.order, i)
	}

	delete(oss.members, aString)
}

// Size returns the number of members in the set
func (oss *OrderedStringSet) Size() int {
	return len(oss.members)
}

// Subset returns true if all set members in this set exist in setB (oss ⊆ setB)
func (oss *OrderedStringSet) Subset(setB *OrderedStringSet) bool {
	for m := range oss.members {
		if _, ok := setB.members[m]; !ok {
			return false
		}
	}

	return true
}

// Union returns a set with members in this set or setB
func (oss *OrderedStringSet) Union(setB *OrderedStringSet) *OrderedStringSet {
	union := NewOrderedStringSet(oss.Elements()...)
	if setB == nil {
		return union
	}

	for _, m := range setB.Elements() {
		union.Add(m)
	}

	return union
}

type stringList struct {
	elements []string
}

func newStringList() *stringList {
	return &stringList{
		elements: make([]string, 0),
	}
}

// at returns the element, true at the given index.
// Empty string, false is returned if there's no value at that index.
func (sl *stringList) at(i int) (string, bool) {
	if i > len(sl.elements) - 1 || i < 0 {
		return "", false
	}

	return sl.elements[i], true
}

// delete removes an element at the given index, and returns it.
// Empty string, false is returned if there's no element at that index to delete.
func (sl *stringList) delete(i int) (string, bool) {
	end := len(sl.elements) - 1
	if i > end || i < 0 {
		return "", false
	}

	e := sl.elements[i]
	// Copy values from the deletion point to the left by one
	copy(sl.elements[i:], sl.elements[i+1:])
	// Set last value to empty string
	sl.elements[end] = ""
	// Truncate the slice
	sl.elements = sl.elements[:end]

	return e, true
}

// insert adds an element at the given index.
// This method of insertion avoids creating a new slice, so garbage collection isn't involved.
func (sl *stringList) insert(i int, aString string) {
	// Add a empty string value to the end of the slice, to make room for the new element.
	sl.elements = append(sl.elements, "")
	// Copy values from the insertion point to the right by one
	copy(sl.elements[i+1:], sl.elements[i:])
	// Set the value at the insertion point
	sl.elements[i] = aString
}

// push adds an element to the end of the list
func (sl *stringList) push(aString string) {
	sl.elements = append(sl.elements, aString)
}

// pop removes an element from the end of the list, and returns it.
// Empty string, false is returned if there's no element to pop.
func (sl *stringList) pop() (string, bool) {
	size := len(sl.elements)
	if size == 0 {
		return "", false
	}

	// This method of deletion is used instead of calling sl.Delete(), because it's faster.
	end := size - 1
	e := sl.elements[end]
	sl.elements[end] = ""
	sl.elements = sl.elements[0:end]

	return e, true
}

// shift removes an element from the front of the list, and returns it.
// Empty string, false is returned if there's no element to shift.
func (sl *stringList) shift() (string, bool) {
	if len(sl.elements) == 0 {
		return "", false
	}

	// This method of deletion is used instead of calling sl.Delete(), because it's faster.
	e := sl.elements[0]
	sl.elements[0] = ""
	sl.elements = sl.elements[1:]
	return e, true
}

// size returns the size of the list
func (sl *stringList) size() int {
	return len(sl.elements)
}

// unshift adds an element to the front of the list
func (sl *stringList) unshift(aString string) {
	sl.insert(0, aString)
}
