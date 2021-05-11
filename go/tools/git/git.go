/*
 *
 * Copyright 2021 The Vitess Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * /
 */

package git

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

type Release struct {
	Name       string
	CommitHash string
}

func GetCommitHashFromClonedRef(ref, repo string) (hash string, err error) {
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           repo,
		ReferenceName: plumbing.ReferenceName(ref),
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.NoTags,
	})
	if err != nil {
		return "", err
	}
	head, err := r.Head()
	if err != nil {
		return "", err
	}
	return head.Hash().String(), nil
}

// GetLatestVitessCommitHash gets the latest Vitess commit hash on master
func GetLatestVitessCommitHash() (hash string, err error) {
	return GetCommitHashFromClonedRef("refs/heads/master", "https://github.com/vitessio/vitess")
}

// GetAllVitessReleaseCommitHash gets all the vitess releases and the commit hashes
func GetAllVitessReleaseCommitHash() ([]*Release, error) {
	repo := "https://github.com/vitessio/vitess"
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:   repo,
		Depth: 1,
	})
	if err != nil {
		return nil, err
	}
	tagrefs, err := r.Tags()
	if err != nil {
		return nil, err
	}

	regexPattern := `^refs/tags/v\d+\.\d+\.\d+$`
	var res []*Release

	err = tagrefs.ForEach(func(t *plumbing.Reference) error {
		tagName := t.Name().String()
		commitHash := t.Hash().String()
		isMatched, err := regexp.MatchString(regexPattern, tagName)
		if isMatched {
			res = append(res, &Release{
				Name:       tagName[11:],
				CommitHash: commitHash,
			})
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetLastReleaseAndCommitHash gets the last release number along with the commit hash
func GetLastReleaseAndCommitHash() (*Release, error) {
	res, err := GetAllVitessReleaseCommitHash()
	if err != nil {
		return nil, err
	}
	maxVersion := res[0]
	for _, release := range res {
		comp, err := compareReleaseNumbers(maxVersion.Name, release.Name)
		if err != nil {
			return nil, err
		}
		if comp == -1 {
			maxVersion = release
		}
	}
	return maxVersion, nil
}

// compareReleaseNumbers compares the two release numbers provided as input
// the result is as follows -
// 0, if release1 == release2
// 1, if release1 > release2
// -1, if release1 < release2
func compareReleaseNumbers(release1string, release2string string) (int, error) {
	release1, err := getVersionNumbersFromString(release1string)
	if err != nil {
		return 0, err
	}
	release2, err := getVersionNumbersFromString(release2string)
	if err != nil {
		return 0, err
	}

	index := 0
	for index < len(release1) && index < len(release2) {
		if release1[index] > release2[index] {
			return 1, nil
		}
		if release1[index] < release2[index] {
			return -1, nil
		}
		index++
	}
	if len(release1) > len(release2) {
		return 1, nil
	}
	if len(release1) < len(release2) {
		return -1, nil
	}
	return 0, nil
}

// getVersionNumbersFromString gets the version numbers as an integer slice from the string provided.
func getVersionNumbersFromString(s string) ([]int, error) {
	tmp := strings.Split(s, ".")
	values := make([]int, 0, len(tmp))
	for _, raw := range tmp {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

func GetCommitHash(repoDir string) (hash string, err error) {
	r, err := git.PlainOpen(repoDir)
	if err != nil {
		return "", err
	}

	ref, err := r.Head()
	if err != nil {
		return "", err
	}

	hash = ref.Hash().String()
	return hash, nil
}

// ShortenSHA will return the first 7 characters of a SHA.
// If the given SHA is too short, it will be returned untouched.
func ShortenSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
