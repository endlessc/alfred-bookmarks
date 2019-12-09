package bookmark

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/pierrec/lz4"
)

func TestFirefoxBookmarks(t *testing.T) {
	tests := []struct {
		description  string
		bookmarkPath string
		want         Bookmarks
		expectErr    bool
	}{
		{
			description:  "correct bookmark file",
			bookmarkPath: "test-firefox-bookmarks.jsonlz4",
			want: Bookmarks{
				&Bookmark{
					Folder: "/Bookmark Menu",
					Title:  "Google",
					Domain: "www.google.com",
					URI:    "https://www.google.com/",
				},
				&Bookmark{
					Folder: "/Bookmark Menu/1-hierarchy-a",
					Title:  "GitHub",
					Domain: "github.com",
					URI:    "https://github.com/",
				},
				&Bookmark{
					Folder: "/Bookmark Menu/1-hierarchy-a/2-hierarchy-a/3-hierarchy-a",
					Title:  "Stack Overflow",
					Domain: "stackoverflow.com",
					URI:    "https://stackoverflow.com/",
				},
				&Bookmark{
					Folder: "/Bookmark Menu/1-hierarchy-a/2-hierarchy-a/3-hierarchy-a",
					Title:  "Amazon Web Services",
					Domain: "aws.amazon.com",
					URI:    "https://aws.amazon.com/?nc1=h_ls",
				},
				&Bookmark{
					Folder: "/Bookmark Menu/1-hierarchy-b",
					Title:  "Yahoo",
					Domain: "www.yahoo.com",
					URI:    "https://www.yahoo.com/",
				},
				&Bookmark{
					Folder: "/Bookmark Menu/1-hierarchy-b/2-hierarchy-a",
					Title:  "Facebook",
					Domain: "www.facebook.com",
					URI:    "https://www.facebook.com/",
				},
				&Bookmark{
					Folder: "/Bookmark Menu/1-hierarchy-b/2-hierarchy-a",
					Title:  "Twitter",
					Domain: "twitter.com",
					URI:    "https://twitter.com/login",
				},
				&Bookmark{
					Folder: "/Bookmark Menu/1-hierarchy-b/2-hierarchy-b",
					Title:  "Amazon.com",
					Domain: "www.amazon.com",
					URI:    "https://www.amazon.com/",
				},
			},
			expectErr: false,
		},
		{
			description:  "invalid bookmark file",
			bookmarkPath: "test",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			b := firefoxBookmark{
				bookmarkPath: tt.bookmarkPath,
			}

			bookmarks, err := b.Bookmarks()
			if tt.expectErr && err == nil {
				t.Errorf("expect error happens, but got response")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error got: %+v", err.Error())
			}

			diff := DiffBookmark(bookmarks, tt.want)
			if !tt.expectErr && diff != "" {
				t.Errorf("unexpected response: (+want -got)\n%+v", diff)
			}
		})
	}
}

// return json string from .jsonlz4
func readDefaultFirefoxBookmarksJSON() (string, error) {
	path, err := GetFirefoxBookmarkFile()
	if err != nil {
		return "", err
	}

	b := firefoxBookmark{
		bookmarkPath:         path,
		firefoxBookmarkEntry: firefoxBookmarkEntry{},
	}

	err = b.unmarshal()
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(b.firefoxBookmarkEntry)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func readTestFirefoxBookmarkJSON() (string, error) {
	jsonData, err := ioutil.ReadFile("test-firefox-bookmarks.json")

	return string(jsonData), err
}

//Create Jsonlz4 from jsonfile
func TestCreateTestFirefoxJsonlz4(t *testing.T) {
	// switch readDefaultFirefoxBookmarksJSON or readTestFirefoxBookmarkJSON
	_, _ = readDefaultFirefoxBookmarksJSON()
	str, err := readTestFirefoxBookmarkJSON()
	if err != nil {
		t.Fatal(err)
	}

	r := strings.NewReader(str)
	w, err := os.Create("test-firefox-bookmarks.jsonlz4")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = compress(r, w, len(str))
	if err != nil {
		t.Logf("Failed to compress data: %s\n", err)
		t.Fail()
		return
	}

}

func compress(src io.Reader, dst io.Writer, intendedSize int) error {
	const magicHeader = "mozLz40\x00"
	_, err := dst.Write([]byte(magicHeader))
	if err != nil {
		return fmt.Errorf("couldn't Write header: %w", err)
	}

	b, err := ioutil.ReadAll(src)
	if err != nil {
		return fmt.Errorf("couldn't ReadAll to Compress: %w", err)
	}

	err = binary.Write(dst, binary.LittleEndian, uint32(intendedSize))
	if err != nil {
		return fmt.Errorf("couldn't encode length: %w", err)
	}

	dstBytes := make([]byte, 10*len(b))
	sz, err := lz4.CompressBlockHC(b, dstBytes, -1)
	if err != nil {
		return fmt.Errorf("couldn't CompressBlock: %w", err)
	}
	if sz == 0 {
		return errors.New("data incompressible")
	}

	_, err = dst.Write(dstBytes[:sz])
	if err != nil {
		return fmt.Errorf("couldn't Write compressed data: %w", err)
	}

	return nil
}
