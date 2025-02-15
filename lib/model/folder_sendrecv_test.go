// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/db"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/fs"
	"github.com/syncthing/syncthing/lib/ignore"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/scanner"
	"github.com/syncthing/syncthing/lib/sync"
)

var blocks = []protocol.BlockInfo{
	{Hash: []uint8{0xfa, 0x43, 0x23, 0x9b, 0xce, 0xe7, 0xb9, 0x7c, 0xa6, 0x2f, 0x0, 0x7c, 0xc6, 0x84, 0x87, 0x56, 0xa, 0x39, 0xe1, 0x9f, 0x74, 0xf3, 0xdd, 0xe7, 0x48, 0x6d, 0xb3, 0xf9, 0x8d, 0xf8, 0xe4, 0x71}}, // Zero'ed out block
	{Offset: 0, Size: 0x20000, Hash: []uint8{0x7e, 0xad, 0xbc, 0x36, 0xae, 0xbb, 0xcf, 0x74, 0x43, 0xe2, 0x7a, 0x5a, 0x4b, 0xb8, 0x5b, 0xce, 0xe6, 0x9e, 0x1e, 0x10, 0xf9, 0x8a, 0xbc, 0x77, 0x95, 0x2, 0x29, 0x60, 0x9e, 0x96, 0xae, 0x6c}},
	{Offset: 131072, Size: 0x20000, Hash: []uint8{0x3c, 0xc4, 0x20, 0xf4, 0xb, 0x2e, 0xcb, 0xb9, 0x5d, 0xce, 0x34, 0xa8, 0xc3, 0x92, 0xea, 0xf3, 0xda, 0x88, 0x33, 0xee, 0x7a, 0xb6, 0xe, 0xf1, 0x82, 0x5e, 0xb0, 0xa9, 0x26, 0xa9, 0xc0, 0xef}},
	{Offset: 262144, Size: 0x20000, Hash: []uint8{0x76, 0xa8, 0xc, 0x69, 0xd7, 0x5c, 0x52, 0xfd, 0xdf, 0x55, 0xef, 0x44, 0xc1, 0xd6, 0x25, 0x48, 0x4d, 0x98, 0x48, 0x4d, 0xaa, 0x50, 0xf6, 0x6b, 0x32, 0x47, 0x55, 0x81, 0x6b, 0xed, 0xee, 0xfb}},
	{Offset: 393216, Size: 0x20000, Hash: []uint8{0x44, 0x1e, 0xa4, 0xf2, 0x8d, 0x1f, 0xc3, 0x1b, 0x9d, 0xa5, 0x18, 0x5e, 0x59, 0x1b, 0xd8, 0x5c, 0xba, 0x7d, 0xb9, 0x8d, 0x70, 0x11, 0x5c, 0xea, 0xa1, 0x57, 0x4d, 0xcb, 0x3c, 0x5b, 0xf8, 0x6c}},
	{Offset: 524288, Size: 0x20000, Hash: []uint8{0x8, 0x40, 0xd0, 0x5e, 0x80, 0x0, 0x0, 0x7c, 0x8b, 0xb3, 0x8b, 0xf7, 0x7b, 0x23, 0x26, 0x28, 0xab, 0xda, 0xcf, 0x86, 0x8f, 0xc2, 0x8a, 0x39, 0xc6, 0xe6, 0x69, 0x59, 0x97, 0xb6, 0x1a, 0x43}},
	{Offset: 655360, Size: 0x20000, Hash: []uint8{0x38, 0x8e, 0x44, 0xcb, 0x30, 0xd8, 0x90, 0xf, 0xce, 0x7, 0x4b, 0x58, 0x86, 0xde, 0xce, 0x59, 0xa2, 0x46, 0xd2, 0xf9, 0xba, 0xaf, 0x35, 0x87, 0x38, 0xdf, 0xd2, 0xd, 0xf9, 0x45, 0xed, 0x91}},
	{Offset: 786432, Size: 0x20000, Hash: []uint8{0x32, 0x28, 0xcd, 0xf, 0x37, 0x21, 0xe5, 0xd4, 0x1e, 0x58, 0x87, 0x73, 0x8e, 0x36, 0xdf, 0xb2, 0x70, 0x78, 0x56, 0xc3, 0x42, 0xff, 0xf7, 0x8f, 0x37, 0x95, 0x0, 0x26, 0xa, 0xac, 0x54, 0x72}},
	{Offset: 917504, Size: 0x20000, Hash: []uint8{0x96, 0x6b, 0x15, 0x6b, 0xc4, 0xf, 0x19, 0x18, 0xca, 0xbb, 0x5f, 0xd6, 0xbb, 0xa2, 0xc6, 0x2a, 0xac, 0xbb, 0x8a, 0xb9, 0xce, 0xec, 0x4c, 0xdb, 0x78, 0xec, 0x57, 0x5d, 0x33, 0xf9, 0x8e, 0xaf}},
}

var folders = []string{"default"}

var diffTestData = []struct {
	a string
	b string
	s int
	d []protocol.BlockInfo
}{
	{"contents", "contents", 1024, []protocol.BlockInfo{}},
	{"", "", 1024, []protocol.BlockInfo{}},
	{"contents", "contents", 3, []protocol.BlockInfo{}},
	{"contents", "cantents", 3, []protocol.BlockInfo{{Offset: 0, Size: 3}}},
	{"contents", "contants", 3, []protocol.BlockInfo{{Offset: 3, Size: 3}}},
	{"contents", "cantants", 3, []protocol.BlockInfo{{Offset: 0, Size: 3}, {Offset: 3, Size: 3}}},
	{"contents", "", 3, []protocol.BlockInfo{{Offset: 0, Size: 0}}},
	{"", "contents", 3, []protocol.BlockInfo{{Offset: 0, Size: 3}, {Offset: 3, Size: 3}, {Offset: 6, Size: 2}}},
	{"con", "contents", 3, []protocol.BlockInfo{{Offset: 3, Size: 3}, {Offset: 6, Size: 2}}},
	{"contents", "con", 3, nil},
	{"contents", "cont", 3, []protocol.BlockInfo{{Offset: 3, Size: 1}}},
	{"cont", "contents", 3, []protocol.BlockInfo{{Offset: 3, Size: 3}, {Offset: 6, Size: 2}}},
}

func setupFile(filename string, blockNumbers []int) protocol.FileInfo {
	// Create existing file
	existingBlocks := make([]protocol.BlockInfo, len(blockNumbers))
	for i := range blockNumbers {
		existingBlocks[i] = blocks[blockNumbers[i]]
	}

	return protocol.FileInfo{
		Name:   filename,
		Blocks: existingBlocks,
	}
}

func createFile(t *testing.T, name string, fs fs.Filesystem) protocol.FileInfo {
	t.Helper()

	f, err := fs.Create(name)
	must(t, err)
	f.Close()
	fi, err := fs.Stat(name)
	must(t, err)
	file, err := scanner.CreateFileInfo(fi, name, fs)
	must(t, err)
	return file
}

func setupSendReceiveFolder(files ...protocol.FileInfo) (*model, *sendReceiveFolder) {
	w := createTmpWrapper(defaultCfg)
	model := newModel(w, myID, "syncthing", "dev", db.OpenMemory(), nil)
	fcfg := testFolderConfigTmp()
	model.AddFolder(fcfg)

	f := &sendReceiveFolder{
		folder: folder{
			stateTracker:        newStateTracker("default", model.evLogger),
			model:               model,
			fset:                model.folderFiles[fcfg.ID],
			initialScanFinished: make(chan struct{}),
			ctx:                 context.TODO(),
			FolderConfiguration: fcfg,
		},

		queue:         newJobQueue(),
		pullErrors:    make(map[string]string),
		pullErrorsMut: sync.NewMutex(),
	}
	f.fs = fs.NewMtimeFS(f.Filesystem(), db.NewNamespacedKV(model.db, "mtime"))

	// Update index
	if files != nil {
		f.updateLocalsFromScanning(files)
	}

	// Folders are never actually started, so no initial scan will be done
	close(f.initialScanFinished)

	return model, f
}

func cleanupSRFolder(f *sendReceiveFolder, m *model) {
	m.evLogger.Stop()
	os.Remove(m.cfg.ConfigPath())
	os.Remove(f.Filesystem().URI())
}

// Layout of the files: (indexes from the above array)
// 12345678 - Required file
// 02005008 - Existing file (currently in the index)
// 02340070 - Temp file on the disk

func TestHandleFile(t *testing.T) {
	// After the diff between required and existing we should:
	// Copy: 2, 5, 8
	// Pull: 1, 3, 4, 6, 7

	existingBlocks := []int{0, 2, 0, 0, 5, 0, 0, 8}
	existingFile := setupFile("filex", existingBlocks)
	requiredFile := existingFile
	requiredFile.Blocks = blocks[1:]

	m, f := setupSendReceiveFolder(existingFile)
	defer cleanupSRFolder(f, m)

	copyChan := make(chan copyBlocksState, 1)
	dbUpdateChan := make(chan dbUpdateJob, 1)

	f.handleFile(requiredFile, copyChan, dbUpdateChan)

	// Receive the results
	toCopy := <-copyChan

	if len(toCopy.blocks) != 8 {
		t.Errorf("Unexpected count of copy blocks: %d != 8", len(toCopy.blocks))
	}

	for _, block := range blocks[1:] {
		found := false
		for _, toCopyBlock := range toCopy.blocks {
			if string(toCopyBlock.Hash) == string(block.Hash) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Did not find block %s", block.String())
		}
	}
}

func TestHandleFileWithTemp(t *testing.T) {
	// After diff between required and existing we should:
	// Copy: 2, 5, 8
	// Pull: 1, 3, 4, 6, 7

	// After dropping out blocks already on the temp file we should:
	// Copy: 5, 8
	// Pull: 1, 6

	existingBlocks := []int{0, 2, 0, 0, 5, 0, 0, 8}
	existingFile := setupFile("file", existingBlocks)
	requiredFile := existingFile
	requiredFile.Blocks = blocks[1:]

	m, f := setupSendReceiveFolder(existingFile)
	defer cleanupSRFolder(f, m)

	if _, err := prepareTmpFile(f.Filesystem()); err != nil {
		t.Fatal(err)
	}

	copyChan := make(chan copyBlocksState, 1)
	dbUpdateChan := make(chan dbUpdateJob, 1)

	f.handleFile(requiredFile, copyChan, dbUpdateChan)

	// Receive the results
	toCopy := <-copyChan

	if len(toCopy.blocks) != 4 {
		t.Errorf("Unexpected count of copy blocks: %d != 4", len(toCopy.blocks))
	}

	for _, idx := range []int{1, 5, 6, 8} {
		found := false
		block := blocks[idx]
		for _, toCopyBlock := range toCopy.blocks {
			if string(toCopyBlock.Hash) == string(block.Hash) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Did not find block %s", block.String())
		}
	}
}

func TestCopierFinder(t *testing.T) {
	// After diff between required and existing we should:
	// Copy: 1, 2, 3, 4, 6, 7, 8
	// Since there is no existing file, nor a temp file

	// After dropping out blocks found locally:
	// Pull: 1, 5, 6, 8

	tempFile := fs.TempName("file2")

	existingBlocks := []int{0, 2, 3, 4, 0, 0, 7, 0}
	existingFile := setupFile(fs.TempName("file"), existingBlocks)
	requiredFile := existingFile
	requiredFile.Blocks = blocks[1:]
	requiredFile.Name = "file2"

	m, f := setupSendReceiveFolder(existingFile)
	defer cleanupSRFolder(f, m)

	if _, err := prepareTmpFile(f.Filesystem()); err != nil {
		t.Fatal(err)
	}

	copyChan := make(chan copyBlocksState)
	pullChan := make(chan pullBlockState, 4)
	finisherChan := make(chan *sharedPullerState, 1)
	dbUpdateChan := make(chan dbUpdateJob, 1)

	// Run a single fetcher routine
	go f.copierRoutine(copyChan, pullChan, finisherChan)

	f.handleFile(requiredFile, copyChan, dbUpdateChan)

	pulls := []pullBlockState{<-pullChan, <-pullChan, <-pullChan, <-pullChan}
	finish := <-finisherChan

	select {
	case <-pullChan:
		t.Fatal("Pull channel has data to be read")
	case <-finisherChan:
		t.Fatal("Finisher channel has data to be read")
	default:
	}

	// Verify that the right blocks went into the pull list.
	// They are pulled in random order.
	for _, idx := range []int{1, 5, 6, 8} {
		found := false
		block := blocks[idx]
		for _, pulledBlock := range pulls {
			if string(pulledBlock.block.Hash) == string(block.Hash) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Did not find block %s", block.String())
		}
		if string(finish.file.Blocks[idx-1].Hash) != string(blocks[idx].Hash) {
			t.Errorf("Block %d mismatch: %s != %s", idx, finish.file.Blocks[idx-1].String(), blocks[idx].String())
		}
	}

	// Verify that the fetched blocks have actually been written to the temp file
	blks, err := scanner.HashFile(context.TODO(), f.Filesystem(), tempFile, protocol.MinBlockSize, nil, false)
	if err != nil {
		t.Log(err)
	}

	for _, eq := range []int{2, 3, 4, 7} {
		if string(blks[eq-1].Hash) != string(blocks[eq].Hash) {
			t.Errorf("Block %d mismatch: %s != %s", eq, blks[eq-1].String(), blocks[eq].String())
		}
	}
	finish.fd.Close()
}

func TestWeakHash(t *testing.T) {
	// Setup the model/pull environment
	model, fo := setupSendReceiveFolder()
	defer cleanupSRFolder(fo, model)
	ffs := fo.Filesystem()

	tempFile := fs.TempName("weakhash")
	var shift int64 = 10
	var size int64 = 1 << 20
	expectBlocks := int(size / protocol.MinBlockSize)
	expectPulls := int(shift / protocol.MinBlockSize)
	if shift > 0 {
		expectPulls++
	}

	f, err := ffs.Create("weakhash")
	must(t, err)
	defer f.Close()
	_, err = io.CopyN(f, rand.Reader, size)
	if err != nil {
		t.Error(err)
	}
	info, err := f.Stat()
	if err != nil {
		t.Error(err)
	}

	// Create two files, second file has `shifted` bytes random prefix, yet
	// both are of the same length, for example:
	// File 1: abcdefgh
	// File 2: xyabcdef
	f.Seek(0, os.SEEK_SET)
	existing, err := scanner.Blocks(context.TODO(), f, protocol.MinBlockSize, size, nil, true)
	if err != nil {
		t.Error(err)
	}

	f.Seek(0, os.SEEK_SET)
	remainder := io.LimitReader(f, size-shift)
	prefix := io.LimitReader(rand.Reader, shift)
	nf := io.MultiReader(prefix, remainder)
	desired, err := scanner.Blocks(context.TODO(), nf, protocol.MinBlockSize, size, nil, true)
	if err != nil {
		t.Error(err)
	}

	existingFile := protocol.FileInfo{
		Name:       "weakhash",
		Blocks:     existing,
		Size:       size,
		ModifiedS:  info.ModTime().Unix(),
		ModifiedNs: int32(info.ModTime().Nanosecond()),
	}
	desiredFile := protocol.FileInfo{
		Name:      "weakhash",
		Size:      size,
		Blocks:    desired,
		ModifiedS: info.ModTime().Unix() + 1,
	}

	fo.updateLocalsFromScanning([]protocol.FileInfo{existingFile})

	copyChan := make(chan copyBlocksState)
	pullChan := make(chan pullBlockState, expectBlocks)
	finisherChan := make(chan *sharedPullerState, 1)
	dbUpdateChan := make(chan dbUpdateJob, 1)

	// Run a single fetcher routine
	go fo.copierRoutine(copyChan, pullChan, finisherChan)

	// Test 1 - no weak hashing, file gets fully repulled (`expectBlocks` pulls).
	fo.WeakHashThresholdPct = 101
	fo.handleFile(desiredFile, copyChan, dbUpdateChan)

	var pulls []pullBlockState
	for len(pulls) < expectBlocks {
		select {
		case pull := <-pullChan:
			pulls = append(pulls, pull)
		case <-time.After(10 * time.Second):
			t.Errorf("timed out, got %d pulls expected %d", len(pulls), expectPulls)
		}
	}
	finish := <-finisherChan

	select {
	case <-pullChan:
		t.Fatal("Pull channel has data to be read")
	case <-finisherChan:
		t.Fatal("Finisher channel has data to be read")
	default:
	}

	finish.fd.Close()
	if err := ffs.Remove(tempFile); err != nil {
		t.Fatal(err)
	}

	// Test 2 - using weak hash, expectPulls blocks pulled.
	fo.WeakHashThresholdPct = -1
	fo.handleFile(desiredFile, copyChan, dbUpdateChan)

	pulls = pulls[:0]
	for len(pulls) < expectPulls {
		select {
		case pull := <-pullChan:
			pulls = append(pulls, pull)
		case <-time.After(10 * time.Second):
			t.Errorf("timed out, got %d pulls expected %d", len(pulls), expectPulls)
		}
	}

	finish = <-finisherChan
	finish.fd.Close()

	expectShifted := expectBlocks - expectPulls
	if finish.copyOriginShifted != expectShifted {
		t.Errorf("did not copy %d shifted", expectShifted)
	}
}

// Test that updating a file removes its old blocks from the blockmap
func TestCopierCleanup(t *testing.T) {
	iterFn := func(folder, file string, index int32) bool {
		return true
	}

	// Create a file
	file := setupFile("test", []int{0})
	m, f := setupSendReceiveFolder(file)
	defer cleanupSRFolder(f, m)

	file.Blocks = []protocol.BlockInfo{blocks[1]}
	file.Version = file.Version.Update(myID.Short())
	// Update index (removing old blocks)
	f.updateLocalsFromScanning([]protocol.FileInfo{file})

	if m.finder.Iterate(folders, blocks[0].Hash, iterFn) {
		t.Error("Unexpected block found")
	}

	if !m.finder.Iterate(folders, blocks[1].Hash, iterFn) {
		t.Error("Expected block not found")
	}

	file.Blocks = []protocol.BlockInfo{blocks[0]}
	file.Version = file.Version.Update(myID.Short())
	// Update index (removing old blocks)
	f.updateLocalsFromScanning([]protocol.FileInfo{file})

	if !m.finder.Iterate(folders, blocks[0].Hash, iterFn) {
		t.Error("Unexpected block found")
	}

	if m.finder.Iterate(folders, blocks[1].Hash, iterFn) {
		t.Error("Expected block not found")
	}
}

func TestDeregisterOnFailInCopy(t *testing.T) {
	file := setupFile("filex", []int{0, 2, 0, 0, 5, 0, 0, 8})

	m, f := setupSendReceiveFolder()
	defer cleanupSRFolder(f, m)

	// Set up our evet subscription early
	s := m.evLogger.Subscribe(events.ItemFinished)

	// queue.Done should be called by the finisher routine
	f.queue.Push("filex", 0, time.Time{})
	f.queue.Pop()

	if f.queue.lenProgress() != 1 {
		t.Fatal("Expected file in progress")
	}

	copyChan := make(chan copyBlocksState)
	pullChan := make(chan pullBlockState)
	finisherBufferChan := make(chan *sharedPullerState)
	finisherChan := make(chan *sharedPullerState)
	dbUpdateChan := make(chan dbUpdateJob, 1)

	go f.copierRoutine(copyChan, pullChan, finisherBufferChan)
	go f.finisherRoutine(finisherChan, dbUpdateChan, make(chan string))

	f.handleFile(file, copyChan, dbUpdateChan)

	// Receive a block at puller, to indicate that at least a single copier
	// loop has been performed.
	toPull := <-pullChan

	// Close the file, causing errors on further access
	toPull.sharedPullerState.fail(os.ErrNotExist)

	// Unblock copier
	go func() {
		for range pullChan {
		}
	}()

	select {
	case state := <-finisherBufferChan:
		// At this point the file should still be registered with both the job
		// queue, and the progress emitter. Verify this.
		if f.model.progressEmitter.lenRegistry() != 1 || f.queue.lenProgress() != 1 || f.queue.lenQueued() != 0 {
			t.Fatal("Could not find file")
		}

		// Pass the file down the real finisher, and give it time to consume
		finisherChan <- state

		t0 := time.Now()
		if ev, err := s.Poll(time.Minute); err != nil {
			t.Fatal("Got error waiting for ItemFinished event:", err)
		} else if n := ev.Data.(map[string]interface{})["item"]; n != state.file.Name {
			t.Fatal("Got ItemFinished event for wrong file:", n)
		}
		t.Log("event took", time.Since(t0))

		state.mut.Lock()
		stateFd := state.fd
		state.mut.Unlock()
		if stateFd != nil {
			t.Fatal("File not closed?")
		}

		if f.model.progressEmitter.lenRegistry() != 0 || f.queue.lenProgress() != 0 || f.queue.lenQueued() != 0 {
			t.Fatal("Still registered", f.model.progressEmitter.lenRegistry(), f.queue.lenProgress(), f.queue.lenQueued())
		}

		// Doing it again should have no effect
		finisherChan <- state

		if _, err := s.Poll(time.Second); err != events.ErrTimeout {
			t.Fatal("Expected timeout, not another event", err)
		}

		if f.model.progressEmitter.lenRegistry() != 0 || f.queue.lenProgress() != 0 || f.queue.lenQueued() != 0 {
			t.Fatal("Still registered", f.model.progressEmitter.lenRegistry(), f.queue.lenProgress(), f.queue.lenQueued())
		}

	case <-time.After(time.Second):
		t.Fatal("Didn't get anything to the finisher")
	}
}

func TestDeregisterOnFailInPull(t *testing.T) {
	file := setupFile("filex", []int{0, 2, 0, 0, 5, 0, 0, 8})

	m, f := setupSendReceiveFolder()
	defer cleanupSRFolder(f, m)

	// Set up our evet subscription early
	s := m.evLogger.Subscribe(events.ItemFinished)

	// queue.Done should be called by the finisher routine
	f.queue.Push("filex", 0, time.Time{})
	f.queue.Pop()

	if f.queue.lenProgress() != 1 {
		t.Fatal("Expected file in progress")
	}

	copyChan := make(chan copyBlocksState)
	pullChan := make(chan pullBlockState)
	finisherBufferChan := make(chan *sharedPullerState)
	finisherChan := make(chan *sharedPullerState)
	dbUpdateChan := make(chan dbUpdateJob, 1)

	go f.copierRoutine(copyChan, pullChan, finisherBufferChan)
	go f.pullerRoutine(pullChan, finisherBufferChan)
	go f.finisherRoutine(finisherChan, dbUpdateChan, make(chan string))

	f.handleFile(file, copyChan, dbUpdateChan)

	// Receive at finisher, we should error out as puller has nowhere to pull
	// from.
	timeout = time.Second
	select {
	case state := <-finisherBufferChan:
		// At this point the file should still be registered with both the job
		// queue, and the progress emitter. Verify this.
		if f.model.progressEmitter.lenRegistry() != 1 || f.queue.lenProgress() != 1 || f.queue.lenQueued() != 0 {
			t.Fatal("Could not find file")
		}

		// Pass the file down the real finisher, and give it time to consume
		finisherChan <- state

		t0 := time.Now()
		if ev, err := s.Poll(time.Minute); err != nil {
			t.Fatal("Got error waiting for ItemFinished event:", err)
		} else if n := ev.Data.(map[string]interface{})["item"]; n != state.file.Name {
			t.Fatal("Got ItemFinished event for wrong file:", n)
		}
		t.Log("event took", time.Since(t0))

		state.mut.Lock()
		stateFd := state.fd
		state.mut.Unlock()
		if stateFd != nil {
			t.Fatal("File not closed?")
		}

		if f.model.progressEmitter.lenRegistry() != 0 || f.queue.lenProgress() != 0 || f.queue.lenQueued() != 0 {
			t.Fatal("Still registered", f.model.progressEmitter.lenRegistry(), f.queue.lenProgress(), f.queue.lenQueued())
		}

		// Doing it again should have no effect
		finisherChan <- state

		if _, err := s.Poll(time.Second); err != events.ErrTimeout {
			t.Fatal("Expected timeout, not another event", err)
		}

		if f.model.progressEmitter.lenRegistry() != 0 || f.queue.lenProgress() != 0 || f.queue.lenQueued() != 0 {
			t.Fatal("Still registered", f.model.progressEmitter.lenRegistry(), f.queue.lenProgress(), f.queue.lenQueued())
		}
	case <-time.After(time.Second):
		t.Fatal("Didn't get anything to the finisher")
	}
}

func TestIssue3164(t *testing.T) {
	m, f := setupSendReceiveFolder()
	defer cleanupSRFolder(f, m)
	ffs := f.Filesystem()
	tmpDir := ffs.URI()

	ignDir := filepath.Join("issue3164", "oktodelete")
	subDir := filepath.Join(ignDir, "foobar")
	must(t, ffs.MkdirAll(subDir, 0777))
	must(t, ioutil.WriteFile(filepath.Join(tmpDir, subDir, "file"), []byte("Hello"), 0644))
	must(t, ioutil.WriteFile(filepath.Join(tmpDir, ignDir, "file"), []byte("Hello"), 0644))
	file := protocol.FileInfo{
		Name: "issue3164",
	}

	matcher := ignore.New(ffs)
	must(t, matcher.Parse(bytes.NewBufferString("(?d)oktodelete"), ""))
	f.ignores = matcher

	dbUpdateChan := make(chan dbUpdateJob, 1)

	f.deleteDir(file, dbUpdateChan, make(chan string))

	if _, err := ffs.Stat("issue3164"); !fs.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestDiff(t *testing.T) {
	for i, test := range diffTestData {
		a, _ := scanner.Blocks(context.TODO(), bytes.NewBufferString(test.a), test.s, -1, nil, false)
		b, _ := scanner.Blocks(context.TODO(), bytes.NewBufferString(test.b), test.s, -1, nil, false)
		_, d := blockDiff(a, b)
		if len(d) != len(test.d) {
			t.Fatalf("Incorrect length for diff %d; %d != %d", i, len(d), len(test.d))
		} else {
			for j := range test.d {
				if d[j].Offset != test.d[j].Offset {
					t.Errorf("Incorrect offset for diff %d block %d; %d != %d", i, j, d[j].Offset, test.d[j].Offset)
				}
				if d[j].Size != test.d[j].Size {
					t.Errorf("Incorrect length for diff %d block %d; %d != %d", i, j, d[j].Size, test.d[j].Size)
				}
			}
		}
	}
}

func BenchmarkDiff(b *testing.B) {
	testCases := make([]struct{ a, b []protocol.BlockInfo }, 0, len(diffTestData))
	for _, test := range diffTestData {
		a, _ := scanner.Blocks(context.TODO(), bytes.NewBufferString(test.a), test.s, -1, nil, false)
		b, _ := scanner.Blocks(context.TODO(), bytes.NewBufferString(test.b), test.s, -1, nil, false)
		testCases = append(testCases, struct{ a, b []protocol.BlockInfo }{a, b})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			blockDiff(tc.a, tc.b)
		}
	}
}

func TestDiffEmpty(t *testing.T) {
	emptyCases := []struct {
		a    []protocol.BlockInfo
		b    []protocol.BlockInfo
		need int
		have int
	}{
		{nil, nil, 0, 0},
		{[]protocol.BlockInfo{{Offset: 3, Size: 1}}, nil, 0, 0},
		{nil, []protocol.BlockInfo{{Offset: 3, Size: 1}}, 1, 0},
	}
	for _, emptyCase := range emptyCases {
		h, n := blockDiff(emptyCase.a, emptyCase.b)
		if len(h) != emptyCase.have {
			t.Errorf("incorrect have: %d != %d", len(h), emptyCase.have)
		}
		if len(n) != emptyCase.need {
			t.Errorf("incorrect have: %d != %d", len(h), emptyCase.have)
		}
	}
}

// TestDeleteIgnorePerms checks, that a file gets deleted when the IgnorePerms
// option is true and the permissions do not match between the file on disk and
// in the db.
func TestDeleteIgnorePerms(t *testing.T) {
	m, f := setupSendReceiveFolder()
	defer cleanupSRFolder(f, m)
	ffs := f.Filesystem()
	f.IgnorePerms = true

	name := "deleteIgnorePerms"
	file, err := ffs.Create(name)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()

	stat, err := file.Stat()
	must(t, err)
	fi, err := scanner.CreateFileInfo(stat, name, ffs)
	must(t, err)
	ffs.Chmod(name, 0600)
	scanChan := make(chan string)
	finished := make(chan struct{})
	go func() {
		err = f.checkToBeDeleted(fi, scanChan)
		close(finished)
	}()
	select {
	case <-scanChan:
		<-finished
	case <-finished:
	}
	must(t, err)
}

func TestCopyOwner(t *testing.T) {
	// Verifies that owner and group are copied from the parent, for both
	// files and directories.

	if runtime.GOOS == "windows" {
		t.Skip("copying owner not supported on Windows")
	}

	const (
		expOwner = 1234
		expGroup = 5678
	)

	// Set up a folder with the CopyParentOwner bit and backed by a fake
	// filesystem.

	m, f := setupSendReceiveFolder()
	defer cleanupSRFolder(f, m)
	f.folder.FolderConfiguration = config.NewFolderConfiguration(m.id, f.ID, f.Label, fs.FilesystemTypeFake, "/TestCopyOwner")
	f.folder.FolderConfiguration.CopyOwnershipFromParent = true

	f.fs = f.Filesystem()

	// Create a parent dir with a certain owner/group.

	f.fs.Mkdir("foo", 0755)
	f.fs.Lchown("foo", expOwner, expGroup)

	dir := protocol.FileInfo{
		Name:        "foo/bar",
		Type:        protocol.FileInfoTypeDirectory,
		Permissions: 0755,
	}

	// Have the folder create a subdirectory, verify that it's the correct
	// owner/group.

	dbUpdateChan := make(chan dbUpdateJob, 1)
	defer close(dbUpdateChan)
	f.handleDir(dir, dbUpdateChan, nil)
	<-dbUpdateChan // empty the channel for later

	info, err := f.fs.Lstat("foo/bar")
	if err != nil {
		t.Fatal("Unexpected error (dir):", err)
	}
	if info.Owner() != expOwner || info.Group() != expGroup {
		t.Fatalf("Expected dir owner/group to be %d/%d, not %d/%d", expOwner, expGroup, info.Owner(), info.Group())
	}

	// Have the folder create a file, verify it's the correct owner/group.
	// File is zero sized to avoid having to handle copies/pulls.

	file := protocol.FileInfo{
		Name:        "foo/bar/baz",
		Type:        protocol.FileInfoTypeFile,
		Permissions: 0644,
	}

	// Wire some stuff. The flow here is handleFile() -[copierChan]->
	// copierRoutine() -[finisherChan]-> finisherRoutine() -[dbUpdateChan]->
	// back to us and we're done. The copier routine doesn't do anything,
	// but it's the way data is passed around. When the database update
	// comes the finisher is done.

	finisherChan := make(chan *sharedPullerState)
	defer close(finisherChan)
	copierChan := make(chan copyBlocksState)
	defer close(copierChan)
	go f.copierRoutine(copierChan, nil, finisherChan)
	go f.finisherRoutine(finisherChan, dbUpdateChan, nil)
	f.handleFile(file, copierChan, nil)
	<-dbUpdateChan

	info, err = f.fs.Lstat("foo/bar/baz")
	if err != nil {
		t.Fatal("Unexpected error (file):", err)
	}
	if info.Owner() != expOwner || info.Group() != expGroup {
		t.Fatalf("Expected file owner/group to be %d/%d, not %d/%d", expOwner, expGroup, info.Owner(), info.Group())
	}

	// Have the folder create a symlink. Verify it accordingly.
	symlink := protocol.FileInfo{
		Name:          "foo/bar/sym",
		Type:          protocol.FileInfoTypeSymlink,
		Permissions:   0644,
		SymlinkTarget: "over the rainbow",
	}

	f.handleSymlink(symlink, dbUpdateChan, nil)
	<-dbUpdateChan

	info, err = f.fs.Lstat("foo/bar/sym")
	if err != nil {
		t.Fatal("Unexpected error (file):", err)
	}
	if info.Owner() != expOwner || info.Group() != expGroup {
		t.Fatalf("Expected symlink owner/group to be %d/%d, not %d/%d", expOwner, expGroup, info.Owner(), info.Group())
	}
}

// TestSRConflictReplaceFileByDir checks that a conflict is created when an existing file
// is replaced with a directory and versions are conflicting
func TestSRConflictReplaceFileByDir(t *testing.T) {
	m, f := setupSendReceiveFolder()
	defer cleanupSRFolder(f, m)
	ffs := f.Filesystem()

	name := "foo"

	// create local file
	file := createFile(t, name, ffs)
	file.Version = protocol.Vector{}.Update(myID.Short())
	f.updateLocalsFromScanning([]protocol.FileInfo{file})

	// Simulate remote creating a dir with the same name
	file.Type = protocol.FileInfoTypeDirectory
	rem := device1.Short()
	file.Version = protocol.Vector{}.Update(rem)
	file.ModifiedBy = rem

	dbUpdateChan := make(chan dbUpdateJob, 1)
	scanChan := make(chan string, 1)

	f.handleDir(file, dbUpdateChan, scanChan)

	if confls := existingConflicts(name, ffs); len(confls) != 1 {
		t.Fatal("Expected one conflict, got", len(confls))
	} else if scan := <-scanChan; confls[0] != scan {
		t.Fatal("Expected request to scan", confls[0], "got", scan)
	}
}

// TestSRConflictReplaceFileByLink checks that a conflict is created when an existing file
// is replaced with a link and versions are conflicting
func TestSRConflictReplaceFileByLink(t *testing.T) {
	m, f := setupSendReceiveFolder()
	defer cleanupSRFolder(f, m)
	ffs := f.Filesystem()

	name := "foo"

	// create local file
	file := createFile(t, name, ffs)
	file.Version = protocol.Vector{}.Update(myID.Short())
	f.updateLocalsFromScanning([]protocol.FileInfo{file})

	// Simulate remote creating a symlink with the same name
	file.Type = protocol.FileInfoTypeSymlink
	file.SymlinkTarget = "bar"
	rem := device1.Short()
	file.Version = protocol.Vector{}.Update(rem)
	file.ModifiedBy = rem

	dbUpdateChan := make(chan dbUpdateJob, 1)
	scanChan := make(chan string, 1)

	f.handleSymlink(file, dbUpdateChan, scanChan)

	if confls := existingConflicts(name, ffs); len(confls) != 1 {
		t.Fatal("Expected one conflict, got", len(confls))
	} else if scan := <-scanChan; confls[0] != scan {
		t.Fatal("Expected request to scan", confls[0], "got", scan)
	}
}
