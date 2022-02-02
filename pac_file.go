package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

var HEADER_MAGIC = [16]byte{70, 105, 108, 101, 80, 97, 99, 72, 101, 97, 100, 101, 114, 65, 0, 0}

type Header struct {
	Magic        [16]byte
	Dword10      uint32
	CreationDate uint32
	PreloadBytes uint32 // aka folders+files data
	Dword1c      uint32
	FoldersNum   uint32 // 3 dwords
	FilesNum     uint32 // 3 dwords
	Dword28      uint32
	FilesXorKey  uint32
	XorBlockSize uint32
}

type FolderRaw struct {
	Seek uint32
	Unk  uint32
	Size uint32
	Name [256]byte
}
type FileRaw struct {
	Seek uint32
	Unk  uint32
	Size uint32
	Name [32]byte
}

type Entry struct {
	Seek     uint32
	FolderId uint32
	Size     uint32
	Name     string
}

// Performs first type of xor
func name_xor_single(key uint32, additive byte, input []byte) string {
	// Endian agnostic
	array := make([]byte, 4)
	binary.LittleEndian.PutUint32(array, key)

	out_len := 0
	out := make([]byte, len(input))
	for i := range input {
		out[i] = input[i] ^ array[i%4]
		array[i%4] += additive
		if out[i] == 0 {
			out_len = i
			break
		}
	}
	return string(out[:out_len])
}

func byte_xor(key uint32, additive byte, header_dword30 uint32, input []byte) []byte {
	// Endian agnostic
	array := make([]byte, 4)
	binary.LittleEndian.PutUint32(array, key)

	// First round
	out := append([]byte{}, input...)
	first_round := int(header_dword30)
	if first_round > len(input) {
		first_round = len(input)
	}
	for i := 0; i < first_round; i++ {
		out[i] = input[i] ^ array[i%4]
		array[i%4] += additive
	}

	// Second round
	v9 := len(input) - (int(header_dword30) * 2)
	if v9 > 0 {
		for i := 0; i < int(header_dword30); i++ {
			out[int(header_dword30)+v9+i] = input[int(header_dword30)+v9+i] ^ array[i%4]
			array[i%4] += additive
		}
	}

	return out
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		filename := args[0]
		f, err := os.OpenFile(filename, os.O_RDONLY, 0666)
		if err != nil {
			fmt.Printf("Error opening '%s' | %v", filename, err)
			return
		}

		var header Header
		err = binary.Read(f, binary.LittleEndian, &header)
		if err != nil {
			fmt.Println("Error reading header:", err)
			return
		}

		fmt.Printf("%+v\n", header)

		// This is made sure so I don't need to rewrite name_xor_single cuz I'm hella lazy ngl
		if header.XorBlockSize <= 256 {
			panic(fmt.Errorf("XorBlockSize = %d <= 256", header.XorBlockSize))
		}

		preload_bytes := make([]byte, header.PreloadBytes-0x34)
		f.Read(preload_bytes[:])
		preload := bytes.NewReader(preload_bytes)

		folders := make([]Entry, header.FoldersNum)
		if len(folders) != 0 {
			for i := range folders {
				var tmp FolderRaw
				err = binary.Read(preload, binary.LittleEndian, &tmp)
				if err != nil {
					panic(err)
				}
				folders[i] = Entry{
					Seek:     tmp.Seek,
					FolderId: tmp.Unk,
					Size:     tmp.Size,
					Name:     name_xor_single(header.CreationDate, byte(tmp.Size), tmp.Name[:]),
				}
			}
		}
		files := make([]Entry, header.FilesNum)
		if len(files) != 0 {
			for i := range files {
				var tmp FileRaw
				err = binary.Read(preload, binary.LittleEndian, &tmp)
				if err != nil {
					panic(err)
				}
				files[i] = Entry{
					Seek:     tmp.Seek,
					FolderId: tmp.Unk,
					Size:     tmp.Size,
					Name:     name_xor_single(header.CreationDate, byte(tmp.Size), tmp.Name[:]),
				}
			}
		}
		fmt.Printf("Unk20: %#v\n", folders)
		fmt.Printf("Unk24: %#v\n", files)

		fmt.Println("Preload left:", preload.Len())

		out_folder := filename + "_out/"
		for i := range folders {
			err = os.MkdirAll(out_folder+folders[i].Name, 0666)
			if err != nil {
				panic(err)
			}
		}

		for i := range files {
			file := &files[i]
			input := make([]byte, file.Size)
			// Seek doesn't reset wtf?
			// f.Seek(int64(header.Unk18+folders[file.FolderId].Seek+file.Seek), 0)
			n, err := f.ReadAt(input[:], int64(header.PreloadBytes+file.Seek))
			if n != int(file.Size) {
				panic(fmt.Errorf("Error reading file '%s' read %d != size %d", file.Name, n, file.Size))
			}
			if err != nil {
				panic(err)
			}
			output := byte_xor(header.CreationDate, byte(header.FilesXorKey), header.XorBlockSize, input[:])
			os.WriteFile(out_folder+folders[file.FolderId].Name+"/"+file.Name, output, 0666)
		}
	} else {
		fmt.Println("Invalid usage!")
	}
}
