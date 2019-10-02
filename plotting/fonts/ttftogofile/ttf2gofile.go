// convert file to byte array and write to variable go file
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func usage() {
	fmt.Println("USAGE:")
	fmt.Println("> ttf2GoFile <filename>")
}

// reade file to byte array
func fileBytes(filePath string) ([]byte, error) {
	var err error
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

// write to variable go file
func createGoFile(fileName string, dataTTF []byte) error {
	f, err := os.Create(strings.ToLower(fileName) + ".go")
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("package fonts\n\nvar %s = %#v\n",
		strings.Title(fileName), dataTTF))

	return err
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	file := os.Args[1]

	fmt.Println(file[len(file)-5:])
	if strings.ToLower(file[len(file)-5:]) != ".ttf" {
		fmt.Printf("File name %s is not font\n", file)
		os.Exit(1)
	}

	fmt.Printf("Reading %s\n", file)

	dataTTF, err := fileBytes(file)

	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}

	if len(dataTTF) == 0 {
		fmt.Printf("File was empty.\n")
		os.Exit(1)
	}

	fmt.Printf("Create file: %s.go\n", file[:len(file)-4])
	err = createGoFile(file[:len(file)-4], dataTTF)
	if err != nil {
		fmt.Println("Go file not created")
		os.Exit(1)
	}
}
