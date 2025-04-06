package main

import (
	"context"
	"fmt"
	"github.com/carlmjohnson/requests"
	"path/filepath"
	"strings"
)

func listBranches() error {
	var result string
	err := requests.
		URL(serverUri).Path("branches").
		ToString(&result).Fetch(context.Background())
	if err == nil {
		fmt.Println(result)
	}
	return err
}

func createBranch(name string) error {
	return requests.
		URL(serverUri).Path("branches").Param("name", name).
		Post().Fetch(context.Background())
}

func listPackages(branch string) error {
	var result string
	err := requests.
		URL(serverUri).Pathf("packages/%s", branch).
		ToString(&result).Fetch(context.Background())
	if err == nil {
		fmt.Println(result)
	}
	return err
}

func getPackage(branch string, name string) error {
	return requests.
		URL(serverUri).Pathf("packages/%s", branch).Param("name", name).
		ToFile(name).Fetch(context.Background())
}

func putPackages(branch string, names []string) error {
	for _, name := range names {
		err := requests.
			URL(serverUri).Pathf("packages/%s", branch).
			Param("name", filepath.Base(name)).
			BodyFile(name).
			Fetch(context.Background())
		if err != nil {
			return err
		}
	}
	return nil
}

func rmPackages(branch string, orgNames []string) error {
	var names []string
	for _, name := range orgNames {
		names = append(names, filepath.Base(name))
	}
	param := strings.Join(names, ",")
	return requests.
		URL(serverUri).Pathf("packages/%s", branch).Param("name", param).
		Delete().Fetch(context.Background())
}
