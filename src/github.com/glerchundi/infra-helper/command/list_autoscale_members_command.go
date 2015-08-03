package command

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/codegangsta/cli"
	"github.com/glerchundi/infra-helper/providers"
	"github.com/glerchundi/infra-helper/providers/aws"
)

func NewListAutoscaleMembersCommand() cli.Command {
	return cli.Command{
		Name:  "list-autoscale-members",
		Flags: []cli.Flag {
			cli.StringFlag{
				Name: "name, n",
				Usage: "search by name",
			},
			cli.StringFlag{
				Name: "prefix",
				Usage: "prepends a static raw string",
			},
			cli.StringFlag{
				Name: "suffix",
				Usage: "appends a static raw string",
			},
			cli.StringFlag{
				Name: "format, f",
				Value: "{{.Name}}={{.Address}}",
				Usage: "defines how to format members output",
			},
			cli.StringFlag{
				Name: "join-separator, s",
				Value: "\\n",
				Usage: "which separator will be used to join members",
			},
			cli.StringFlag{
				Name: "out, o",
				Usage: "save output to a file",
			},
		},
		Action: handleListAutoscaleMembers,
	}
}

func handleListAutoscaleMembers(c *cli.Context) {
	name := c.String("name")
	prefix := c.String("prefix")
	suffix := c.String("suffix")
	formatTemplate := c.String("format")
	joinSeparator := c.String("join-separator")
	outputFilePath := c.String("out")

	tmpl, err := template.New("format").Parse(formatTemplate)
	if err != nil {
		log.Fatal(err)
	}

	// sanitize join separator
	joinSeparator = strings.Replace(joinSeparator, "\\n", "\n", -1)
	joinSeparator = strings.Replace(joinSeparator, "\\r", "\r", -1)
	joinSeparator = strings.Replace(joinSeparator, "\\t", "\t", -1)

	var provider providers.Provider = aws.New()

    // retrieve cluster members
	var clusterMembersByName map[string]string = map[string]string{
		"H":  "Hydrogen",
		"He": "Helium",
		"Li": "Lithium",
	}

	if name == "" {
		clusterMembersByName, err = provider.GetClusterMembers()
		if err != nil {
			log.Fatal(err)
		}
	} else if false {
		clusterMembersByName, err = provider.GetClusterMembersByName(name)
		if err != nil {
			log.Fatal(err)
		}
	}

	// create intermediate key map
	sortedNames := make([]string, 0)
	for name := range clusterMembersByName {
		sortedNames = append(sortedNames, name)
	}

	// do string sorting
	sort.Strings(sortedNames)

	// loop over sorted name (which are the keys in the map)
	data := ""
	for _, name := range sortedNames {
		// template provisioning object
		nameAndAddress := struct {
			Name string
			Address string
		}{ name, clusterMembersByName[name] }

		// execute template
		line, err := executeTemplate(tmpl, nameAndAddress)
		if err != nil {
			log.Fatal(err)
		}

		// save current processed line
		if data == "" {
			data = fmt.Sprintf("%s", line)
		} else {
			data = fmt.Sprintf("%s%s%s", data, joinSeparator, line)
		}
	}

	// prepend & append, prefix & suffix
	data = fmt.Sprintf("%s%s%s", prefix, data, suffix)

	if outputFilePath == "" {
		printToStdout(data)
	} else {
		printToFile(outputFilePath, data)
	}
}

func getMd5(name string) (string, error) {
	if !isFileExist(name) {
		return "", errors.New("file not found")
	}

	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	io.Copy(h, f)

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func isFileExist(fpath string) bool {
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		return false
	}
	return true
}

func isSameFile(src, dest string) (bool, error) {
	if !isFileExist(dest) {
		return false, nil
	}

	dstMd5, err := getMd5(dest)
	if err != nil {
		return false, err
	}

	srcMd5, err := getMd5(src)
	if err != nil {
		return false, err
	}

	if dstMd5 != srcMd5 {
		log.Print(fmt.Sprintf("%s has md5sum %s should be %s", dest, dstMd5, srcMd5))
		return false, nil
	}

	return true, nil
}

func printToFile(outputFilePath, data string) (err error) {
	tempFilePath := outputFilePath + ".tmp"
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// in case it exits unexpectedly
	isClosed := false
	defer func() { if !isClosed { tempFile.Close() } }()

	// print to file
	tempFile.WriteString(data)

	if err := tempFile.Close(); err != nil {
		log.Fatal(err)
	}
	isClosed = true

	isSameFile, err := isSameFile(tempFilePath, outputFilePath)
	if err != nil {
		log.Print(err)
		return
	}

	if !isSameFile {
		if err := os.Rename(tempFilePath, outputFilePath); err != nil {
			log.Fatal(err)
		}
	} else {
		if err = os.Remove(tempFilePath); err != nil {
			log.Print(err)
			return
		}
	}

	return
}

func printToStdout(data string) (err error) {
	_, err = os.Stdout.WriteString(data)
	return
}

func executeTemplate(tmpl *template.Template, data interface{}) (string, error) {
	var cmdBuffer bytes.Buffer
	if err := tmpl.Execute(&cmdBuffer, data); err != nil {
		return "", err
	}
	return cmdBuffer.String(), nil
}