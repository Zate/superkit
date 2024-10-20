package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	replaceID           = "AABBCCDD"
	bootstrapFolderName = "bootstrap"
	reponame            = "https://github.com/anthdm/superkit.git"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usage()
	}
	var projectPath string
	projectName := args[0]
	installPath := ""

	if args[1] != "" {
		installPath = args[1]
		// validate install path
		if _, err := os.Stat(installPath); errors.Is(err, fs.ErrNotExist) {
			log.Fatal("install path does not exist")
		}
		// check if project folder already exists
		if _, err := os.Stat(path.Join(installPath, projectName)); !os.IsNotExist(err) {
			log.Fatal("project folder already exists")
		}
	}

	projectPath = path.Join(installPath, projectName)

	// check if superkit folder already exists, if so, delete
	_, err := os.Stat("superkit")
	if !os.IsNotExist(err) {
		fmt.Println("-- deleting superkit folder cause its already present")
		cleanUp()
	}

	fmt.Println("-- cloning", reponame)
	clone := exec.Command("git", "clone", reponame)
	if err := clone.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("-- renaming bootstrap ->", projectName)
	if err := os.Rename(path.Join("superkit", bootstrapFolderName), projectPath); err != nil {
		log.Fatal(err)
	}

	err = filepath.Walk(path.Join(projectPath), func(fullPath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		b, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}

		contentStr := string(b)
		if strings.Contains(contentStr, replaceID) {
			replacedContent := strings.ReplaceAll(contentStr, replaceID, projectName)
			file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = file.WriteString(replacedContent)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("-- renaming .env.local -> .env")
	if err := os.Rename(
		path.Join(projectPath, ".env.local"),
		path.Join(projectPath, ".env"),
	); err != nil {
		log.Fatal(err)
	}

	fmt.Println("-- generating secure secret")
	pathToDotEnv := path.Join(projectPath, ".env")
	b, err := os.ReadFile(pathToDotEnv)
	if err != nil {
		log.Fatal(err)
	}
	secret := generateSecret()
	replacedContent := strings.Replace(string(b), "{{app_secret}}", secret, -1)
	file, err := os.OpenFile(pathToDotEnv, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	_, err = file.WriteString(replacedContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("-- project (%s) successfully installed!\n", projectPath)

	// if err := checkDevDependencies(); err != nil {
	// 	fmt.Println(err)
	// }

	cleanUp()
}

func generateSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}

// checkDevDependencies checks if the required dev dependencies are installed
func checkDevDependencies() error {
	deps := []string{"npm", "templ"}
	missingDeps := []string{}
	for _, dep := range deps {
		_, err := exec.LookPath(dep)
		if err != nil {
			missingDeps = append(missingDeps, dep)
			continue
		}
		fmt.Printf("-- %s found\n", dep)
	}
	if len(missingDeps) > 0 {
		return fmt.Errorf("missing dependencies: %v", missingDeps)
		// fmt.Println("Please install the following dependencies:")
		// for _, dep := range missingDeps {
		// 	fmt.Println("\t", dep)
		// }
		// log.Fatal("Exiting...")
		// os.Exit(1)
	}
	return nil
}

// cleanUp removes the superkit folder
func cleanUp() {
	if err := os.RemoveAll("superkit"); err != nil {
		log.Fatal(err)
	}
}

// usage prints the usage of the program
func usage() {
	fmt.Println()
	fmt.Println("install requires your project name as the first argument")
	fmt.Println("with optional path to install the project as the second argument")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("\tgo run superkit/install.go [your_project_name] [optional_path_to_install_project]")
	fmt.Println()
	os.Exit(1)
}
