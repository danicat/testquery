// Copyright (c) 2026 Google LLC
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package collector

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

type moduleContext struct {
	projectRoot string
	modulePath  string
}

func evalSymlinksClean(p string) string {
	abs, err := filepath.Abs(p)
	if err == nil {
		p = abs
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return filepath.Clean(resolved)
	}
	// If the file itself does not exist yet, try resolving its parent directory.
	dir := filepath.Dir(p)
	base := filepath.Base(p)
	if resolvedDir, err := filepath.EvalSymlinks(dir); err == nil {
		return filepath.Join(resolvedDir, base)
	}
	return filepath.Clean(p)
}

func resolveModuleContext(pkgDirs []string) (*moduleContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	var candidates []string
	for _, d := range pkgDirs {
		cleaned := strings.TrimSuffix(strings.TrimSuffix(d, "/..."), "...")
		if cleaned == "" {
			cleaned = "."
		}
		if info, statErr := os.Stat(cleaned); statErr == nil && info.IsDir() {
			candidates = append(candidates, evalSymlinksClean(cleaned))
		}
	}
	candidates = append(candidates, evalSymlinksClean(cwd))

	for _, start := range candidates {
		dir := start
		for {
			goModPath := filepath.Join(dir, "go.mod")
			if data, readErr := os.ReadFile(goModPath); readErr == nil {
				return &moduleContext{
					projectRoot: dir,
					modulePath:  modfile.ModulePath(data),
				}, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	root := evalSymlinksClean(cwd)
	if len(candidates) > 0 {
		root = candidates[0]
	}
	return &moduleContext{
		projectRoot: root,
		modulePath:  "",
	}, nil
}

func (mc *moduleContext) normalizePackage(rawPkg string) string {
	if rawPkg == "" {
		return mc.modulePath
	}
	slashed := filepath.ToSlash(rawPkg)
	if mc.modulePath != "" {
		if slashed == mc.modulePath || strings.HasPrefix(slashed, mc.modulePath+"/") {
			return slashed
		}
	}
	if filepath.IsAbs(rawPkg) && mc.projectRoot != "" {
		absPkg := evalSymlinksClean(rawPkg)
		if rel, err := filepath.Rel(mc.projectRoot, absPkg); err == nil && !strings.HasPrefix(rel, "..") {
			relSlash := filepath.ToSlash(rel)
			if mc.modulePath != "" {
				if relSlash == "." || relSlash == "" {
					return mc.modulePath
				}
				return mc.modulePath + "/" + relSlash
			}
			return relSlash
		}
	}
	trimmed := strings.TrimPrefix(slashed, "./")
	if mc.modulePath != "" {
		if trimmed == "." || trimmed == "" {
			return mc.modulePath
		}
		return mc.modulePath + "/" + trimmed
	}
	return trimmed
}

func (mc *moduleContext) normalizeFileAndPackage(rawFile string, fallbackPkg string) (string, string) {
	slashed := filepath.ToSlash(rawFile)
	var normFile string

	if mc.modulePath != "" && strings.HasPrefix(slashed, mc.modulePath+"/") {
		normFile = strings.TrimPrefix(slashed, mc.modulePath+"/")
	} else if filepath.IsAbs(rawFile) && mc.projectRoot != "" {
		absFile := evalSymlinksClean(rawFile)
		if rel, err := filepath.Rel(mc.projectRoot, absFile); err == nil && !strings.HasPrefix(rel, "..") {
			normFile = filepath.ToSlash(rel)
		} else {
			normFile = filepath.Base(rawFile)
		}
	} else if mc.projectRoot != "" {
		absFile := evalSymlinksClean(filepath.Join(mc.projectRoot, rawFile))
		if _, err := os.Stat(absFile); err == nil {
			if rel, relErr := filepath.Rel(mc.projectRoot, absFile); relErr == nil && !strings.HasPrefix(rel, "..") {
				normFile = filepath.ToSlash(rel)
			}
		}
		if normFile == "" {
			normFile = strings.TrimPrefix(slashed, "./")
		}
	} else {
		normFile = strings.TrimPrefix(slashed, "./")
	}

	normFile = strings.TrimPrefix(normFile, "./")
	relDir := path.Dir(normFile)

	var normPkg string
	if mc.modulePath != "" {
		if relDir == "." || relDir == "" {
			normPkg = mc.modulePath
		} else {
			normPkg = mc.modulePath + "/" + relDir
		}
	} else if fallbackPkg != "" && !strings.HasSuffix(fallbackPkg, ".go") {
		normPkg = mc.normalizePackage(fallbackPkg)
	} else {
		normPkg = relDir
	}

	return normFile, normPkg
}

func (mc *moduleContext) resolveFileOnDisk(rawFile string) (string, error) {
	if filepath.IsAbs(rawFile) {
		if _, err := os.Stat(rawFile); err == nil {
			return rawFile, nil
		}
	}

	normFile, _ := mc.normalizeFileAndPackage(rawFile, "")
	if mc.projectRoot != "" {
		candidate := filepath.Join(mc.projectRoot, filepath.FromSlash(normFile))
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	if _, err := os.Stat(rawFile); err == nil {
		abs, absErr := filepath.Abs(rawFile)
		if absErr == nil {
			return abs, nil
		}
		return rawFile, nil
	}

	if mc.projectRoot != "" {
		targetSuffix := string(filepath.Separator) + filepath.FromSlash(normFile)
		var foundPath string
		err := filepath.Walk(mc.projectRoot, func(p string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !info.IsDir() && (strings.HasSuffix(p, targetSuffix) || filepath.Base(p) == normFile) {
				abs, absErr := filepath.Abs(p)
				if absErr != nil {
					return absErr
				}
				foundPath = abs
				return filepath.SkipDir
			}
			return nil
		})
		if err == nil && foundPath != "" {
			return foundPath, nil
		}
	}

	return "", fmt.Errorf("failed to find file %s in %s", rawFile, mc.projectRoot)
}

func (mc *moduleContext) resolvePackageDir(pkgImportPath string) string {
	if mc.modulePath != "" {
		if pkgImportPath == mc.modulePath {
			return mc.projectRoot
		}
		if after, ok := strings.CutPrefix(pkgImportPath, mc.modulePath+"/"); ok {
			rel := after
			return filepath.Join(mc.projectRoot, filepath.FromSlash(rel))
		}
	}
	if filepath.IsAbs(pkgImportPath) {
		return pkgImportPath
	}
	if mc.projectRoot != "" {
		candidate := filepath.Join(mc.projectRoot, filepath.FromSlash(pkgImportPath))
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "."
}
