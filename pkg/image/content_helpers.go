package image

import (
	"archive/tar"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/tree"
)

func fetchFileContentsByPath(filetree *tree.FileTree, fileCatalog *FileCatalog, path file.Path) (string, error) {
	fileReference := filetree.File(path)
	if fileReference == nil {
		return "", fmt.Errorf("could not find file path in Tree: %s", path)
	}

	// if this is a link resolve to the final file reference...
	var err error
	fileReference, err = resolveLink(*fileReference, filetree, fileCatalog)
	if err != nil {
		return "", err
	}

	content, err := fileCatalog.FileContents(*fileReference)
	if err != nil {
		return "", err
	}
	return content, nil
}

func fetchMultipleFileContentsByPath(filetree *tree.FileTree, fileCatalog *FileCatalog, paths ...file.Path) (map[file.Reference]string, error) {
	fileReferences := make([]file.Reference, len(paths))
	for idx, path := range paths {
		fileReference := filetree.File(path)
		if fileReference == nil {
			return nil, fmt.Errorf("could not find file path in Tree: %s", path)
		}

		// if this is a link resolve to the final file reference...
		var err error
		fileReference, err = resolveLink(*fileReference, filetree, fileCatalog)
		if err != nil {
			return nil, err
		}

		fileReferences[idx] = *fileReference
	}

	content, err := fileCatalog.MultipleFileContents(fileReferences...)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func resolveLink(ref file.Reference, t *tree.FileTree, fileCatalog *FileCatalog) (*file.Reference, error) {
	alreadySeen := file.NewFileReferenceSet()
	currentRef := &ref
	for {
		if alreadySeen.Contains(*currentRef) {
			return nil, fmt.Errorf("cycle during symlink resolution: %+v", currentRef)
		}

		entry, err := fileCatalog.Get(*currentRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve link metadata (%+v): %w", currentRef, err)
		}

		if entry.Metadata.TypeFlag != tar.TypeSymlink && entry.Metadata.TypeFlag != tar.TypeLink {
			// resolved the link to a file!
			return currentRef, nil
		} else if entry.Metadata.Linkname == "" {
			// no resolution and there is no next link (pseudo dead link)... return what you found
			// any content fetches will fail, but that's ok
			return currentRef, nil
		}

		// prepare for the next iteration
		alreadySeen.Add(*currentRef)

		var nextPath string
		if strings.HasPrefix(entry.Metadata.Linkname, "/") {
			// use links with absolute paths blindly
			nextPath = entry.Metadata.Linkname
		} else {
			// resolve relative link paths
			var parentDir string
			switch entry.Metadata.TypeFlag {
			case tar.TypeSymlink:
				parentDir, _ = filepath.Split(string(currentRef.Path))
			case tar.TypeLink:
				parentDir = "/"
			default:
				return nil, fmt.Errorf("unknown link type: %+v", entry.Metadata.TypeFlag)
			}

			// assemble relative link path by normalizing: "/cur/dir/../file1.txt" --> "/cur/file1.txt"
			nextPath = filepath.Clean(path.Join(parentDir, entry.Metadata.Linkname))
		}

		nextRef := t.File(file.Path(nextPath))

		// if there is no next path, return this reference (dead link)
		if nextRef == nil {
			return currentRef, nil
		}
		currentRef = nextRef
	}
}
