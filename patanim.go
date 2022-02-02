package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func transformEncoding(rawReader io.Reader, trans transform.Transformer) (string, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(rawReader, trans))
	if err == nil {
		return string(ret), nil
	} else {
		return "", err
	}
}

// Convert a string encoding from ShiftJIS to UTF-8
func FromShiftJIS(str string) (string, error) {
	return transformEncoding(strings.NewReader(str), japanese.ShiftJIS.NewDecoder())
}

const _STR = uint32(1381258079) // start of file

const P_ST = uint32(1414750032) // block start

const PPST = uint32(1414746192)

// ---

const PGST = uint32(1414743888)

const _END = uint32(1145980255) // end of file

func p_st(f io.Reader) {
	var some_size_idk uint32
	binary.Read(f, binary.LittleEndian, &some_size_idk)
	fmt.Printf("\tsome_size_idk: %d\n", some_size_idk)

	var anim [32]byte

	var command uint32
	for {
		for {
			binary.Read(f, binary.LittleEndian, &command)

			// P_ED
			if command == 1145397072 {
				return // ?
			}
			// PANM
			if command != 1296974160 {
				break
			}

			f.Read(anim[:])
			anim_len := 0
			for i := range anim {
				if anim[i] == 0 {
					anim_len = i
					break
				}
			}
			anim_name, _ := FromShiftJIS(string(anim[:anim_len]))
			fmt.Printf("\t\tPANM: '%s'\n", anim_name)
		}

		// PRST
		if command == 1414746704 {
			break
		}
	}

	binary.Read(f, binary.LittleEndian, &some_size_idk)
	fmt.Printf("\tsome_size_idk2: %d\n", some_size_idk)

	for {
		binary.Read(f, binary.LittleEndian, &command)

		// PRRV
		if command == 1448235600 {
			var b [1]byte
			f.Read(b[:])
			fmt.Printf("\t\tPRRV: %d\n", b[0])
		} else
		// PRXY
		if command == 1498960464 {
			var dw2 [2]int32
			binary.Read(f, binary.LittleEndian, &dw2)
			fmt.Printf("\t\tPRXY: %v\n", dw2)
		} else
		// PRPR
		if command == 1380995664 {
			var dw [1]int32
			binary.Read(f, binary.LittleEndian, &dw)
			fmt.Printf("\t\tPRPR: %d\n", dw[0])
		} else
		//PRAN
		if command == 1312903760 {
			var f32 [1]float32
			binary.Read(f, binary.LittleEndian, &f32)
			fmt.Printf("\t\tPRAN: %d\n", f32[0])
		} else
		// PRSP
		if command == 1347637840 {
			var dw [1]int32
			binary.Read(f, binary.LittleEndian, &dw)
			fmt.Printf("\t\tPRSP: %d\n", dw[0])
			// return
		} else
		// PRMZ
		if command == 1297764944 {
			var f32_2 [2]float32
			binary.Read(f, binary.LittleEndian, &f32_2)
			fmt.Printf("\t\tPRMZ: %v\n", f32_2)
		} else
		// PRCL
		if command == 1279480400 {
			var dw [1]int32
			binary.Read(f, binary.LittleEndian, &dw)
			fmt.Printf("\t\tPRCL: %d\n", dw[0])
		} else
		// PRFL
		if command == 1279677008 {
			var b [1]byte
			f.Read(b[:])
			fmt.Printf("\t\tPRFL: %d\n", b[0])
		} else {
			var brih [4]byte
			binary.LittleEndian.PutUint32(brih[:], command)
			fmt.Printf("\t\tUNK COMMAND: '%s' - %d\n", string(brih[:]), command)
			return
		}
	}
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

		var header [32]byte
		f.Read(header[:])
		fmt.Printf("Header: %v\n", header)

		var command uint32
		binary.Read(f, binary.LittleEndian, &command)

		if command == _STR {
			fmt.Println("Command _STR")
			for {
				for {
					for {
						binary.Read(f, binary.LittleEndian, &command)
						if command <= PPST {
							break
						}
						if command == P_ST {
							fmt.Println("Command P_ST")
							p_st(f)
							return
						}
					}
					if command != PPST {
						break
					}
					fmt.Println("Command PPST")
				}
				if command == _END {
					fmt.Println("Command _END")
					break
				}
				if command == PGST {
					fmt.Println("Command PGST")
					return
				}
			}
		}
		fmt.Println("SUCC!")
	} else {
		fmt.Println("Invalid usage!")
	}
}
