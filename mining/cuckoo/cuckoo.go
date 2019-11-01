// Copyright (c) 2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cuckoo

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qitmeer-lib/crypto/cuckoo"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/soterutil"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	DefaultEdgeBits = 19
	DefaultSipHash = 1
	DefaultGPUEdgeBits = 29
)

type Solver interface {
	// CheckRequirements returns an error if any external requirement for the solver isn't found
	CheckRequirements() error

	// Solve runs the solver with given bytes, and nonce.
	// It returns a slice of solutions.
	Solve([]byte, uint32) ([][]uint32, error)

	// Verify verifies the cycle nonces for given bytes
	Verify([]byte, uint32, []uint32) error
}

// OrigCuckoo uses the original cuckoo solver to provide a Solver interface.
type OrigCuckoo struct {
	EdgeBits int
	SipHash int
}

// GPUCuckoo uses the original mean cuckoo solver with NVIDIA CUDA to provide a Solver interface
type GPUCuckoo struct {
	EdgeBits int
	SipHash int
}

// QitmeerCuckaroo uses the qitmeer cuckaroo solver to provide a Solver interface.
type QitmeerCuckaroo struct {
	c *cuckoo.Cuckoo
}

// maxInt returns the largest integer provided
func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// minInt returns the smallest integer provided
func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// dumpToFile writes the bytes to a file, and returns a os.File type and error
func dumpToFile(data []byte, pattern string) (*os.File, error) {
	fh, err := ioutil.TempFile("", pattern)
	if err != nil {
		return nil, err
	}

	_, err = fh.Write(data)
	if err != nil {
		return nil, err
	}

	err = fh.Close()
	if err != nil {
		return nil, err
	}

	return fh, nil
}

// callSolver calls the solver with the given arguments, and returns any solutions found
func callSolver(name string, args ...string) ([][]uint32, error) {
	cmd := exec.Command(name, args...)

	output, err := cmd.Output()
	if err != nil {
		return [][]uint32{}, err
	}

	var solutions = make([][]uint32, 0)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), " ", 2)
		if len(parts) != 2 {
			continue
		} else if parts[0] != "Solution" {
			continue
		}

		cycle := make([]uint32, 0)
		for i, h := range strings.Split(parts[1], " ") {
			nonceBytes, err := hex.DecodeString(fmt.Sprintf("%08s", h))
			if err != nil {
				return [][]uint32{}, fmt.Errorf("Failed to parse solution %d cycle nonce %d value %s: %s",
					len(solutions), i, h, err)
			}

			nonce := binary.BigEndian.Uint32(nonceBytes)

			cycle = append(cycle, nonce)
		}

		if len(cycle) > 0 {
			solutions = append(solutions, cycle)
		}
	}

	return solutions, nil
}

// CallCuckooGPUSolver runs the given cuckoo GPU solver, and returns a slice of any solutions found
func CallCuckooGPUSolver(name string, header []byte, nonce uint32) ([][]uint32, error) {
	// Write the header to a temp file
	pattern := fmt.Sprintf("cuckoogpusolve-%d-*.bin", nonce)
	fh, err := dumpToFile(header, pattern)
	defer func() {
		_ = os.Remove(fh.Name())
	}()
	if err != nil {
		return [][]uint32{}, err
	}

	args := []string{
		// Bytes of the header
		"-i", fh.Name(),
		// Nonce value
		"-n", fmt.Sprintf("%d", nonce),
	}

	return callSolver(name, args...)
}

// CallCuckooSolver runs the given cuckoo solver, and returns a slice of any solutions found
func CallCuckooSolver(name string, header []byte, nonce uint32) ([][]uint32, error) {
	// Cuckoo solver has had issues when run with more than 4 threads
	threadCount := minInt(maxInt(runtime.NumCPU(), 1), 4)

	// Write the header to a temp file
	pattern := fmt.Sprintf("cuckoosolve-%d-*.bin", nonce)
	fh, err := dumpToFile(header, pattern)
	defer func() {
		_ = os.Remove(fh.Name())
	}()
	if err != nil {
		return [][]uint32{}, err
	}

	args := []string{
		// Bytes of the header
		"-i", fh.Name(),
		// Nonce value
		"-n", fmt.Sprintf("%d", nonce),
		// Number of threads to use for solver
		"-t", fmt.Sprintf("%d", threadCount),
	}

	return callSolver(name, args...)
}

func CallCuckooVerify(name string, header []byte, nonce uint32, cycle []uint32) error {
	// Write the header to a temp file
	fh, err := dumpToFile(header, "cuckooverify-*.bin")
	defer func() {
		_ = os.Remove(fh.Name())
	}()
	if err != nil {
		return err
	}

	cmd := exec.Command(name,
		// Bytes of the header
		"-i", fh.Name(),
		// Nonce value
		"-n", fmt.Sprintf("%d", nonce))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	// Write the cycle nonces to the verify command
	var cycleHex = make([]string, len(cycle))
	for i, nonce := range cycle {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, nonce)
		cycleHex[i] = strings.TrimLeft(hex.EncodeToString(b), "0")
	}

	solution := fmt.Sprintf("Solution %s\n", strings.Join(cycleHex, " "))
	_, err = io.WriteString(stdin, solution)
	if err != nil {
		_ = stdin.Close()
		return err
	}

	err = stdin.Close()
	if err != nil {
		return err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		i := strings.Index(line, "Verified with cyclehash")
		if i != -1 {
			// Verification passed
			return nil
		}

		i = strings.Index(line, "FAILED due to")
		if i != -1 {
			// Verification failed
			return fmt.Errorf(line)
		}
	}

	return fmt.Errorf("Verification failed without a specific error message")
}

// DefaultSolver returns a new QitmeerCuckoo type, with default parameters
func DefaultSolver() Solver {
	var solver Solver
	s := NewCuckarooSolver()
	solver = s

	return solver
}

// NewCuckooSolver returns a new OrigCuckoo type, which
// implements the Solver interface.
func NewCuckooSolver(edgeBits, sipHash int) *OrigCuckoo {
	return &OrigCuckoo{
		EdgeBits: edgeBits,
		SipHash:  sipHash,
	}
}

// NewGPUCuckooSolver returns a new GPUCuckoo type, which
// implements the Solver interface
func NewGPUCuckooSolver() *GPUCuckoo {
	return &GPUCuckoo{
		EdgeBits: DefaultGPUEdgeBits,
	}
}

// NewCuckarooSolver returns a new QitmeerCuckoo type, which
// implements the Solver interface.
func NewCuckarooSolver() *QitmeerCuckaroo {
	return &QitmeerCuckaroo{
		c: cuckoo.NewCuckoo(),
	}
}

// solverName returns the name of the solver program
func (s *OrigCuckoo) solverName() string {
	return fmt.Sprintf("lean%dx%d", s.EdgeBits, s.SipHash)
}

// verifierName returns the name of the solution verifier program
func (s *OrigCuckoo) verifierName() string {
	return fmt.Sprintf("verify%d", s.EdgeBits)
}

// CheckRequirements returns an error if any external requirements for the solver weren't found
func (s *OrigCuckoo) CheckRequirements() error {
	for _, name := range []string{s.solverName(), s.verifierName()} {
		_, found := soterutil.Which(name)
		if !found {
			return fmt.Errorf("Couldn't find %s executable", name)
		}
	}

	return nil
}

// Solve calls the original Cuckoo solver.
// It returns a slice of solutions found for the header bytes.
func (s *OrigCuckoo) Solve(header []byte, nonce uint32) ([][]uint32, error) {
	name := s.solverName()
	solver, found := soterutil.Which(name)
	if !found {
		panic(fmt.Sprintf("Couldn't find %s executable", name))
	}
	return CallCuckooSolver(solver, header, nonce)
}

// Verify uses the original cuckoo solver to verify the cycle nonces for the given header bytes and nonce.
func (s *OrigCuckoo) Verify(header []byte, nonce uint32, cycle []uint32) error {
	name := s.verifierName()
	verify, found := soterutil.Which(name)
	if !found {
		panic(fmt.Sprintf("Couldn't find %s executable", name))
	}
	return CallCuckooVerify(verify, header, nonce, cycle)
}

// solverName returns the name of the solver program
func (g *GPUCuckoo) solverName() string {
	return fmt.Sprintf("cuda%d", g.EdgeBits)
}

// verifierName returns the name of the solution verifier program
func (g *GPUCuckoo) verifierName() string {
	return fmt.Sprintf("verify%d", g.EdgeBits)
}

// CheckRequirements returns an error if any external requirements for the solver weren't found
func (g *GPUCuckoo) CheckRequirements() error {
	for _, name := range []string{g.solverName(), g.verifierName()} {
		_, found := soterutil.Which(name)
		if !found {
			return fmt.Errorf("Couldn't find %s executable", name)
		}
	}

	return nil
}

// Solve calls the original cuckoo GPU solver.
// It returns a slice of solutions found for the header bytes.
func (g *GPUCuckoo) Solve(header []byte, nonce uint32) ([][]uint32, error) {
	name := g.solverName()
	solver, found := soterutil.Which(name)
	if !found {
		panic(fmt.Sprintf("Couldn't find %s executable", name))
	}
	return CallCuckooGPUSolver(solver, header, nonce)
}

// Verify uses the original cuckoo GPU solver to verify the cycle nonces for the given header bytes and nonce.
func (g *GPUCuckoo) Verify(header []byte, nonce uint32, cycle []uint32) error {
	name := g.verifierName()
	verify, found := soterutil.Which(name)
	if !found {
		panic(fmt.Sprintf("Couldn't find %s executable", name))
	}
	return CallCuckooVerify(verify, header, nonce, cycle)
}

// CheckRequirements returns an error if any external requirements for the solver weren't found
func (s *QitmeerCuckaroo) CheckRequirements() error {
	// There are no external requirements for qitmeer cuckaroo, aside from importing the module
	return nil
}

// Solve calls the Qitmeer cuckaroo solver
func (s *QitmeerCuckaroo) Solve(header []byte, nonce uint32) ([][]uint32, error) {
	hash := chainhash.DoubleHashH(header)
	hashBytes := hash.CloneBytes()

	cycle, found := s.c.PoW(hashBytes)
	if !found {
		return [][]uint32{}, fmt.Errorf("No solutions found")
	}

	err := cuckoo.Verify(hashBytes, cycle)
	if err != nil {
		return [][]uint32{}, err
	}

	return [][]uint32{cycle}, nil
}

// Verify uses the Qitmeer cuckoo solver to verify the cycle nonces for the given header
func (s *QitmeerCuckaroo) Verify(header []byte, nonce uint32, cycle []uint32) error {
	hash := chainhash.DoubleHashH(header)
	return cuckoo.Verify(hash.CloneBytes(), cycle)
}