// Copyright 2018 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build extended

package scss

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/bep/go-tocss/scss"
	"github.com/bep/go-tocss/scss/libsass"
	"github.com/bep/go-tocss/tocss"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/media"
	"github.com/gohugoio/hugo/resource"
)

// Used in tests. This feature requires Hugo to be built with the extended tag.
func Supports() bool {
	return true
}

func (t *toCSSTransformation) Transform(ctx *resource.ResourceTransformationCtx) error {
	ctx.OutMediaType = media.CSSType

	var outName string
	if t.options.from.TargetPath != "" {
		ctx.OutPath = t.options.from.TargetPath
	} else {
		ctx.ReplaceOutPathExtension(".css")
	}

	outName = path.Base(ctx.OutPath)

	options := t.options

	options.to.IncludePaths = t.c.sfs.RealDirs(path.Dir(ctx.SourcePath))

	// Append any workDir relative include paths
	for _, ip := range options.from.IncludePaths {
		options.to.IncludePaths = append(options.to.IncludePaths, t.c.workFs.RealDirs(filepath.Clean(ip))...)
	}

	if ctx.InMediaType.SubType == media.SASSType.SubType {
		options.to.SassSyntax = true
	}

	if options.from.EnableSourceMap {

		options.to.SourceMapFilename = outName + ".map"
		options.to.SourceMapRoot = t.c.rs.WorkingDir

		// Setting this to the relative input filename will get the source map
		// more correct for the main entry path (main.scss typically), but
		// it will mess up the import mappings. As a workaround, we do a replacement
		// in the source map itself (see below).
		//options.InputPath = inputPath
		options.to.OutputPath = outName
		options.to.SourceMapContents = true
		options.to.OmitSourceMapURL = false
		options.to.EnableEmbeddedSourceMap = false
	}

	res, err := t.c.toCSS(options.to, ctx.To, ctx.From)
	if err != nil {
		return err
	}

	if options.from.EnableSourceMap && res.SourceMapContent != "" {
		sourcePath := t.c.sfs.RealFilename(ctx.SourcePath)

		if strings.HasPrefix(sourcePath, t.c.rs.WorkingDir) {
			sourcePath = strings.TrimPrefix(sourcePath, t.c.rs.WorkingDir+helpers.FilePathSeparator)
		}

		// This needs to be Unix-style slashes, even on Windows.
		// See https://github.com/gohugoio/hugo/issues/4968
		sourcePath = filepath.ToSlash(sourcePath)

		// This is a workaround for what looks like a bug in Libsass. But
		// getting this resolution correct in tools like Chrome Workspaces
		// is important enough to go this extra mile.
		mapContent := strings.Replace(res.SourceMapContent, `stdin",`, fmt.Sprintf("%s\",", sourcePath), 1)

		return ctx.PublishSourceMap(mapContent)
	}
	return nil
}

func (c *Client) toCSS(options scss.Options, dst io.Writer, src io.Reader) (tocss.Result, error) {
	var res tocss.Result

	transpiler, err := libsass.New(options)
	if err != nil {
		return res, err
	}

	res, err = transpiler.Execute(dst, src)
	if err != nil {
		return res, fmt.Errorf("SCSS processing failed: %s", err)
	}

	return res, nil
}
