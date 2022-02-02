package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

var HEADER_MAGIC = [32]byte{66, 77, 80, 32, 67, 117, 116, 116, 101, 114, 51, 0, 0, 0, 0, 0}

// size 0x648???
type BMPCutFileRaw struct {
	Name    [32]byte    // 0x0 - 0x20
	Dword20 uint32      // 0x20 - 0x24
	Width   uint32      // 0x24 - 0x28
	Height  uint32      // 0x28-0x2c
	BPP     uint32      // 0x2c-0x30, bits per pixel?
	Gap24   [16]byte    // 0x2C - 0x40 (0x14)
	Offset  uint32      // 0x40 - 0x44, mul'd by 24
	Count   uint16      // 0x44 - 0x46, mul'd by 24
	Pad     uint16      // 0x46 - 0x48
	Dword48 [256]uint32 // this section only has 256 or 1 or 0 useful u/i32's
}

type BMPCutFile struct {
	Name    string
	Dword20 uint32
	Width   uint32
	Height  uint32
	BPP     uint32

	Offset uint32
	Count  uint16
}

type BMPCutFileDataRaw struct {
	X      uint32 // 0
	Y      uint32 // 4
	Width  uint32 // 8
	Height uint32 // c
	Unk10  uint16 // 10
	Unk12  uint16 // 12
	Unk14  uint16 // 14
	Unk16  byte   // 16
	Pad17  byte   // 17
}

type BMPCutFileData struct {
	Unk0             uint16
	Unk2             uint16
	Unk4             uint32 // var + (var/dw2024) + const_div*var
	Unk8             int8
	CopyFromPrevious uint8

	CutImage uint16
	CutUnk10 uint16
	CutUnk12 uint16
	Offset   int64
}

type Header struct {
	Magic          [16]byte
	Dword10        uint32
	Unk14          [0x400]byte
	Gap            [0x1c10]byte // wtf???
	Dword2024      uint32       // 0x2024
	Dword2         uint32       // 0x2028
	Dword3         uint32       // 0x202c
	Dword4         uint32       // 0x2030
	Dword5         uint32       // 0x2034
	Dword6         uint32       // 0x2038
	Dword7         uint32       // 0x203c
	Dword8         uint32       // 0x2040
	FileOffsets    [3000]uint32 // 0x2044-0x4f24
	FileDataOffset uint32       // 0x4f24
	Unk            uint32       // 0x4f28
	TotalFileSize  uint32       // 0x4f2c
}

func main() {
	args := os.Args[1:]
	if len(args) > 1 {
		filename := args[0]
		filename_palette := args[1]
		f, err := os.OpenFile(filename, os.O_RDONLY, 0666)
		if err != nil {
			fmt.Printf("Error opening '%s' | %v", filename, err)
			return
		}
		p, err := os.OpenFile(filename_palette, os.O_RDONLY, 0666)
		if err != nil {
			fmt.Printf("Error opening palette '%s' | %v", filename, err)
			return
		}

		// real quick pallete parse?
		var palette_count uint32
		binary.Read(p, binary.LittleEndian, &palette_count)
		palettes := make([][256]uint32, palette_count)
		binary.Read(p, binary.LittleEndian, &palettes)

		var header Header
		err = binary.Read(f, binary.LittleEndian, &header)
		if err != nil {
			fmt.Println("Error reading header:", err)
			return
		}

		// I think I won't print header?
		fmt.Printf("{Magic:%v, Dwords: %v, FileDataOffset:%X, Unk:%X, TotalFileSize:%x}\n", header.Magic, [...]uint32{header.Dword2024, header.Dword2, header.Dword3, header.Dword4, header.Dword5, header.Dword6, header.Dword7, header.Dword8}, header.FileDataOffset, header.Unk, header.TotalFileSize)

		fmt.Printf("%x V 4f30\n", header.FileOffsets[0])
		num_files := 0
		for _, v := range header.FileOffsets {
			if v != 0xFFFFFFFF {
				num_files += 1
			}
		}
		fmt.Printf("Num files: %d\n", num_files)

		counters := [4]int{0, 0, 0, 0}

		out_dir := filename + "_out/"
		err = os.MkdirAll(out_dir, 0666)
		if err != nil {
			panic(err)
		}

		cut_files_data := make([][]BMPCutFileData, num_files)
		cut_files_idx := 0
		for i, v := range header.FileOffsets {
			if v == 0xFFFFFFFF {
				continue
			}

			var cut BMPCutFileRaw
			_, err := f.Seek(int64(v), 0)
			if err != nil {
				panic(fmt.Errorf("Error seeking cut of %d | %v", i, err))
			}
			err = binary.Read(f, binary.LittleEndian, &cut)
			if err != nil {
				panic(fmt.Errorf("Error reading cut of %d | %v", i, err))
			}
			name_len := 0
			for i := range cut.Name {
				if cut.Name[i] == 0 {
					name_len = i
					break
				}
			}
			cut_f := BMPCutFile{
				Name:    string(cut.Name[:name_len]),
				Dword20: cut.Dword20,
				Width:   cut.Width,
				Height:  cut.Height,
				BPP:     cut.BPP,
				Offset:  cut.Offset,
				Count:   cut.Count,
			}
			fmt.Printf("%+v\n", cut_f)

			// ---
			counters[0] += 1 // valid file
			counters[1] += int(cut.Count)

			seek_data := int64(header.FileDataOffset + (cut.Offset * 24))
			f.Seek(seek_data, 0)
			data := make([]BMPCutFileData, int(cut.Count))
			for j := range data {
				var tmp BMPCutFileDataRaw
				err = binary.Read(f, binary.LittleEndian, &tmp)
				if err != nil {
					panic(fmt.Errorf("Error reading data of %d[%d] | %v", i, j, err))
				}
				fmt.Printf("\t%+v\n", tmp)

				// Debug counter stuff
				switch cut.Dword20 {
				case 2:
				case 4:
					counters[2] += 256
					break
				case 3:
					counters[2] += 1
					break
				}
				v12 := 0
				switch cut.Dword20 {
				case 0:
				case 2:
				case 3:
					v12 = int(tmp.Width * tmp.Height)
					break
				case 1:
					v12 = int(4 * tmp.Width * tmp.Height)
					break
				case 4:
				case 5:
					v12 = int(2 * tmp.Width * tmp.Height)
					break
				default:
					break
				}
				counters[3] += v12

				// ---
				data[j].Unk0 = uint16(tmp.X)
				data[j].Unk2 = uint16(tmp.Y)
				if tmp.Height < tmp.Width {
					data[j].Unk8 = -int8(tmp.Width / header.Dword2024)
				} else {
					data[j].Unk8 = int8(tmp.Height / header.Dword2024)
				}
				data[j].CopyFromPrevious = tmp.Unk16

				v29 := uint32(tmp.Unk10) / header.Dword2024
				v30 := (0x10000 / (header.Dword2024 * header.Dword2024)) * uint32(tmp.Unk14)
				data[j].Unk4 = v30 + (uint32(tmp.Unk12) / header.Dword2024) + ((0x100 / header.Dword2024) * v29)

				data[j].CutImage = tmp.Unk14
				data[j].CutUnk10 = tmp.Unk12
				data[j].CutUnk12 = tmp.Unk14

				fmt.Printf("\t\t%+v\n", data[j])
			}
			// ---
			v36 := 0
			bmpcut_arr_offset := int(v) + 0x48
			if cut.BPP != 0 {
				if cut.Dword20 != 2 {
					if cut.Dword20 == 3 {
						v36 = 1
						// label_30 aka copy u32 array of size v36 to somewhere else
						bmpcut_arr_offset += v36 * 4
					} else if cut.Dword20 != 4 {
						// ???
						// cut.Dword48[0] = -1
					} else {
						v36 = 256
						// label_30
						bmpcut_arr_offset += v36 * 4
					}
				} else {
					v36 = 256
					// label_30
					bmpcut_arr_offset += v36 * 4
				}
				// start from label_31

				fmt.Printf("\t%X\n", bmpcut_arr_offset)

				img := image.NewRGBA(image.Rect(0, 0, int(cut.Width), int(cut.Height)))
				parsed := false
				_, err := f.Seek(int64(bmpcut_arr_offset), 0)
				if err != nil {
					panic(err)
				}
				// if cut.BPP == 8 {
				// 	// XXXX -> RGBA?
				// 	parsed = true
				// 	for block_idx, block := range data {
				// 		x_offset := int(block.Unk0)
				// 		y_offset := int(block.Unk2)

				// 		width := int(header.Dword2024)
				// 		height := width * int(block.Unk8)
				// 		if block.Unk8 < 0 {
				// 			height = int(header.Dword2024)
				// 			width = height * int(-block.Unk8)
				// 		}

				// 		image_data := make([]byte, width*height)
				// 		pos, _ := f.Seek(0, 1)
				// 		data[block_idx].Offset = pos
				// 		fmt.Printf("\t\t%X [%dx%d] %d\n", pos, width, height, block.CopyFromPrevious)
				// 		if block.CopyFromPrevious != 0 {
				// 			// found := false
				// 			// var found_block BMPCutFileData
				// 			found_offset := int64(0)
				// 			for jj := cut_files_idx; jj >= 0; jj-- {
				// 				found := false
				// 				for _, bblock := range cut_files_data[jj] {
				// 					if bblock.CopyFromPrevious == 0 {
				// 						// if (bblock.Unk0 == block.Unk0) && (bblock.Unk2 == block.Unk2) && (bblock.CutImage == block.CutImage) {
				// 						if (bblock.Unk0 == block.Unk0) && (bblock.Unk2 == block.Unk2) && (bblock.CutUnk10 == block.CutUnk10) && (bblock.CutUnk12 == block.CutUnk12) && (bblock.CutImage == block.CutImage) {
				// 							// if (bblock.Unk0 == block.Unk0) && (bblock.Unk2 == block.Unk2) && (bblock.CutImage == block.CutImage) {
				// 							found = true
				// 							// found_block = bblock
				// 							found_offset = bblock.Offset
				// 							break
				// 						}
				// 					}
				// 				}
				// 				if found {
				// 					break
				// 				}
				// 			}
				// 			if found_offset != 0 {
				// 				fmt.Printf("\t\t\t%X [%dx%d]\n", found_offset, width, height)
				// 				f.Seek(found_offset, 0)
				// 				err = binary.Read(f, binary.LittleEndian, &image_data)
				// 				if err != nil {
				// 					panic(err)
				// 				}
				// 				for y := 0; y < height; y++ {
				// 					for x := 0; x < width; x++ {
				// 						img.SetRGBA(x+x_offset, y+y_offset, color.RGBA{
				// 							B: image_data[(y*width)+x],
				// 							G: image_data[(y*width)+x],
				// 							R: image_data[(y*width)+x],
				// 							A: image_data[(y*width)+x],
				// 						})
				// 					}
				// 				}
				// 				f.Seek(pos, 0)
				// 			}
				// 		} else {
				// 			err = binary.Read(f, binary.LittleEndian, &image_data)
				// 			if err != nil {
				// 				panic(err)
				// 			}

				// 			for y := 0; y < height; y++ {
				// 				for x := 0; x < width; x++ {
				// 					img.SetRGBA(x+x_offset, y+y_offset, color.RGBA{
				// 						B: image_data[(y*width)+x],
				// 						G: image_data[(y*width)+x],
				// 						R: image_data[(y*width)+x],
				// 						A: image_data[(y*width)+x],
				// 					})
				// 				}
				// 			}
				// 		}
				// 	}
				// } else
				if cut.Dword20 == 1 {
					// 32 bit BGRA
					parsed = true
					for block_idx, block := range data {
						x_offset := int(block.Unk0)
						y_offset := int(block.Unk2)

						width := int(header.Dword2024)
						height := width * int(block.Unk8)
						if block.Unk8 < 0 {
							height = int(header.Dword2024)
							width = height * int(-block.Unk8)
						}
						image_data := make([][4]byte, width*height)
						pos, _ := f.Seek(0, 1)
						data[block_idx].Offset = pos
						fmt.Printf("\t\t%X [%dx%d]\n", pos, width, height)
						err = binary.Read(f, binary.LittleEndian, &image_data)
						if err != nil {
							panic(err)
						}

						for y := 0; y < height; y++ {
							for x := 0; x < width; x++ {
								img.SetRGBA(x+x_offset, y+y_offset, color.RGBA{
									B: image_data[(y*width)+x][0],
									G: image_data[(y*width)+x][1],
									R: image_data[(y*width)+x][2],
									A: image_data[(y*width)+x][3],
								})
							}
						}
					}
				}

				if parsed {
					out, err := os.Create(out_dir + cut_f.Name)
					if err != nil {
						panic(err)
					}
					err = png.Encode(out, img)
					if err != nil {
						panic(err)
					}
				}
			} else {
				// ???
				// cut.Dword48[0] = -1
				// cut.Dword48[3] = -1
			}

			cut_files_data[i] = data
			cut_files_idx++
		}

		fmt.Printf("Counters: %v\n", counters)
	} else {
		fmt.Println("Invalid usage!")
	}
}
