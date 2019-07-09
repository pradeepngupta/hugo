// Copyright 2019 The Hugo Authors. All rights reserved.
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

package hugolib

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/resources/page"
	"github.com/stretchr/testify/require"
)

func TestNestedSections(t *testing.T) {

	var (
		assert  = require.New(t)
		cfg, fs = newTestCfg()
		th      = testHelper{cfg, fs, t}
	)

	cfg.Set("permalinks", map[string]string{
		"perm a": ":sections/:title",
	})

	pageTemplate := `---
title: T%d_%d
---
Content
`

	// Home page
	writeSource(t, fs, filepath.Join("content", "_index.md"), fmt.Sprintf(pageTemplate, -1, -1))

	// Top level content page
	writeSource(t, fs, filepath.Join("content", "mypage.md"), fmt.Sprintf(pageTemplate, 1234, 5))

	// Top level section without index content page
	writeSource(t, fs, filepath.Join("content", "top", "mypage2.md"), fmt.Sprintf(pageTemplate, 12345, 6))
	// Just a page in a subfolder, i.e. not a section.
	writeSource(t, fs, filepath.Join("content", "top", "folder", "mypage3.md"), fmt.Sprintf(pageTemplate, 12345, 67))

	for level1 := 1; level1 < 3; level1++ {
		writeSource(t, fs, filepath.Join("content", "l1", fmt.Sprintf("page_1_%d.md", level1)),
			fmt.Sprintf(pageTemplate, 1, level1))
	}

	// Issue #3586
	writeSource(t, fs, filepath.Join("content", "post", "0000.md"), fmt.Sprintf(pageTemplate, 1, 2))
	writeSource(t, fs, filepath.Join("content", "post", "0000", "0001.md"), fmt.Sprintf(pageTemplate, 1, 3))
	writeSource(t, fs, filepath.Join("content", "elsewhere", "0003.md"), fmt.Sprintf(pageTemplate, 1, 4))

	// Empty nested section, i.e. no regular content pages.
	writeSource(t, fs, filepath.Join("content", "empty1", "b", "c", "_index.md"), fmt.Sprintf(pageTemplate, 33, -1))
	// Index content file a the end and in the middle.
	writeSource(t, fs, filepath.Join("content", "empty2", "b", "_index.md"), fmt.Sprintf(pageTemplate, 40, -1))
	writeSource(t, fs, filepath.Join("content", "empty2", "b", "c", "d", "_index.md"), fmt.Sprintf(pageTemplate, 41, -1))

	// Empty with content file in the middle.
	writeSource(t, fs, filepath.Join("content", "empty3", "b", "c", "d", "_index.md"), fmt.Sprintf(pageTemplate, 41, -1))
	writeSource(t, fs, filepath.Join("content", "empty3", "b", "empty3.md"), fmt.Sprintf(pageTemplate, 3, -1))

	// Section with permalink config
	writeSource(t, fs, filepath.Join("content", "perm a", "link", "_index.md"), fmt.Sprintf(pageTemplate, 9, -1))
	for i := 1; i < 4; i++ {
		writeSource(t, fs, filepath.Join("content", "perm a", "link", fmt.Sprintf("page_%d.md", i)),
			fmt.Sprintf(pageTemplate, 1, i))
	}
	writeSource(t, fs, filepath.Join("content", "perm a", "link", "regular", fmt.Sprintf("page_%d.md", 5)),
		fmt.Sprintf(pageTemplate, 1, 5))

	writeSource(t, fs, filepath.Join("content", "l1", "l2", "_index.md"), fmt.Sprintf(pageTemplate, 2, -1))
	writeSource(t, fs, filepath.Join("content", "l1", "l2_2", "_index.md"), fmt.Sprintf(pageTemplate, 22, -1))
	writeSource(t, fs, filepath.Join("content", "l1", "l2", "l3", "_index.md"), fmt.Sprintf(pageTemplate, 3, -1))

	for level2 := 1; level2 < 4; level2++ {
		writeSource(t, fs, filepath.Join("content", "l1", "l2", fmt.Sprintf("page_2_%d.md", level2)),
			fmt.Sprintf(pageTemplate, 2, level2))
	}
	for level2 := 1; level2 < 3; level2++ {
		writeSource(t, fs, filepath.Join("content", "l1", "l2_2", fmt.Sprintf("page_2_2_%d.md", level2)),
			fmt.Sprintf(pageTemplate, 2, level2))
	}
	for level3 := 1; level3 < 3; level3++ {
		writeSource(t, fs, filepath.Join("content", "l1", "l2", "l3", fmt.Sprintf("page_3_%d.md", level3)),
			fmt.Sprintf(pageTemplate, 3, level3))
	}

	writeSource(t, fs, filepath.Join("content", "Spaces in Section", "page100.md"), fmt.Sprintf(pageTemplate, 10, 0))

	writeSource(t, fs, filepath.Join("layouts", "_default", "single.html"), "<html>Single|{{ .Title }}</html>")
	writeSource(t, fs, filepath.Join("layouts", "_default", "list.html"),
		`
{{ $sect := (.Site.GetPage "l1/l2") }}
<html>List|{{ .Title }}|L1/l2-IsActive: {{ .InSection $sect }}
{{ range .Paginator.Pages }}
PAG|{{ .Title }}|{{ $sect.InSection . }}
{{ end }}
{{/* https://github.com/gohugoio/hugo/issues/4989 */}}
{{ $sections := (.Site.GetPage "section" .Section).Sections.ByWeight }}
</html>`)

	cfg.Set("paginate", 2)

	s := buildSingleSite(t, deps.DepsCfg{Fs: fs, Cfg: cfg}, BuildCfg{})

	require.Len(t, s.RegularPages(), 21)

	tests := []struct {
		sections string
		verify   func(assert *require.Assertions, p page.Page)
	}{
		{"elsewhere", func(assert *require.Assertions, p page.Page) {
			assert.Len(p.Pages(), 1)
			for _, p := range p.Pages() {
				assert.Equal("elsewhere", p.SectionsPath())
			}
		}},
		{"post", func(assert *require.Assertions, p page.Page) {
			assert.Len(p.Pages(), 2)
			for _, p := range p.Pages() {
				assert.Equal("post", p.Section())
			}
		}},
		{"empty1", func(assert *require.Assertions, p page.Page) {
			// > b,c
			assert.NotNil(getPage(p, "/empty1/b"))
			assert.NotNil(getPage(p, "/empty1/b/c"))

		}},
		{"empty2", func(assert *require.Assertions, p page.Page) {
			// > b,c,d where b and d have content files.
			b := getPage(p, "/empty2/b")
			assert.NotNil(b)
			assert.Equal("T40_-1", b.Title())
			c := getPage(p, "/empty2/b/c")

			assert.NotNil(c)
			assert.Equal("Cs", c.Title())
			d := getPage(p, "/empty2/b/c/d")

			assert.NotNil(d)
			assert.Equal("T41_-1", d.Title())

			assert.False(c.Eq(d))
			assert.True(c.Eq(c))
			assert.False(c.Eq("asdf"))

		}},
		{"empty3", func(assert *require.Assertions, p page.Page) {
			// b,c,d with regular page in b
			b := getPage(p, "/empty3/b")
			assert.NotNil(b)
			assert.Len(b.Pages(), 1)
			assert.Equal("empty3.md", b.Pages()[0].File().LogicalName())

		}},
		{"empty3", func(assert *require.Assertions, p page.Page) {
			xxx := getPage(p, "/empty3/nil")
			assert.Nil(xxx)
		}},
		{"top", func(assert *require.Assertions, p page.Page) {
			assert.Equal("Tops", p.Title())
			assert.Len(p.Pages(), 2)
			assert.Equal("mypage2.md", p.Pages()[0].File().LogicalName())
			assert.Equal("mypage3.md", p.Pages()[1].File().LogicalName())
			home := p.Parent()
			assert.True(home.IsHome())
			assert.Len(p.Sections(), 0)
			assert.Equal(home, home.CurrentSection())
			active, err := home.InSection(home)
			assert.NoError(err)
			assert.True(active)
			assert.Equal(p, p.FirstSection())
		}},
		{"l1", func(assert *require.Assertions, p page.Page) {
			assert.Equal("L1s", p.Title())
			assert.Len(p.Pages(), 2)
			assert.True(p.Parent().IsHome())
			assert.Len(p.Sections(), 2)
		}},
		{"l1,l2", func(assert *require.Assertions, p page.Page) {
			assert.Equal("T2_-1", p.Title())
			assert.Len(p.Pages(), 3)
			assert.Equal(p, p.Pages()[0].Parent())
			assert.Equal("L1s", p.Parent().Title())
			assert.Equal("/l1/l2/", p.RelPermalink())
			assert.Len(p.Sections(), 1)

			for _, child := range p.Pages() {

				assert.Equal(p, child.CurrentSection())
				active, err := child.InSection(p)
				assert.NoError(err)

				assert.True(active)
				active, err = p.InSection(child)
				assert.NoError(err)
				assert.True(active)
				active, err = p.InSection(getPage(p, "/"))
				assert.NoError(err)
				assert.False(active)

				isAncestor, err := p.IsAncestor(child)
				assert.NoError(err)
				assert.True(isAncestor)
				isAncestor, err = child.IsAncestor(p)
				assert.NoError(err)
				assert.False(isAncestor)

				isDescendant, err := p.IsDescendant(child)
				assert.NoError(err)
				assert.False(isDescendant)
				isDescendant, err = child.IsDescendant(p)
				assert.NoError(err)
				assert.True(isDescendant)
			}

			assert.True(p.Eq(p.CurrentSection()))

		}},
		{"l1,l2_2", func(assert *require.Assertions, p page.Page) {
			assert.Equal("T22_-1", p.Title())
			assert.Len(p.Pages(), 2)
			assert.Equal(filepath.FromSlash("l1/l2_2/page_2_2_1.md"), p.Pages()[0].File().Path())
			assert.Equal("L1s", p.Parent().Title())
			assert.Len(p.Sections(), 0)
		}},
		{"l1,l2,l3", func(assert *require.Assertions, p page.Page) {
			nilp, _ := p.GetPage("this/does/not/exist")

			assert.Equal("T3_-1", p.Title())
			assert.Len(p.Pages(), 2)
			assert.Equal("T2_-1", p.Parent().Title())
			assert.Len(p.Sections(), 0)

			l1 := getPage(p, "/l1")
			isDescendant, err := l1.IsDescendant(p)
			assert.NoError(err)
			assert.False(isDescendant)
			isDescendant, err = l1.IsDescendant(nil)
			assert.NoError(err)
			assert.False(isDescendant)
			isDescendant, err = nilp.IsDescendant(p)
			assert.NoError(err)
			assert.False(isDescendant)
			isDescendant, err = p.IsDescendant(l1)
			assert.NoError(err)
			assert.True(isDescendant)

			isAncestor, err := l1.IsAncestor(p)
			assert.NoError(err)
			assert.True(isAncestor)
			isAncestor, err = p.IsAncestor(l1)
			assert.NoError(err)
			assert.False(isAncestor)
			assert.Equal(l1, p.FirstSection())
			isAncestor, err = p.IsAncestor(nil)
			assert.NoError(err)
			assert.False(isAncestor)
			isAncestor, err = nilp.IsAncestor(l1)
			assert.NoError(err)
			assert.False(isAncestor)

		}},
		{"perm a,link", func(assert *require.Assertions, p page.Page) {
			assert.Equal("T9_-1", p.Title())
			assert.Equal("/perm-a/link/", p.RelPermalink())
			assert.Len(p.Pages(), 4)
			first := p.Pages()[0]
			assert.Equal("/perm-a/link/t1_1/", first.RelPermalink())
			th.assertFileContent("public/perm-a/link/t1_1/index.html", "Single|T1_1")

			last := p.Pages()[3]
			assert.Equal("/perm-a/link/t1_5/", last.RelPermalink())

		}},
	}

	home := s.getPage(page.KindHome)

	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("sections %s", test.sections), func(t *testing.T) {
			t.Parallel()
			assert := require.New(t)
			sections := strings.Split(test.sections, ",")
			p := s.getPage(page.KindSection, sections...)
			assert.NotNil(p, fmt.Sprint(sections))

			if p.Pages() != nil {
				assert.Equal(p.Pages(), p.Data().(page.Data).Pages())
			}
			assert.NotNil(p.Parent(), fmt.Sprintf("Parent nil: %q", test.sections))
			test.verify(assert, p)
		})
	}

	assert.NotNil(home)

	assert.Len(home.Sections(), 9)
	assert.Equal(home.Sections(), s.Info.Sections())

	rootPage := s.getPage(page.KindPage, "mypage.md")
	assert.NotNil(rootPage)
	assert.True(rootPage.Parent().IsHome())

	// Add a odd test for this as this looks a little bit off, but I'm not in the mood
	// to think too hard a out this right now. It works, but people will have to spell
	// out the directory name as is.
	// If we later decide to do something about this, we will have to do some normalization in
	// getPage.
	// TODO(bep)
	sectionWithSpace := s.getPage(page.KindSection, "Spaces in Section")
	require.NotNil(t, sectionWithSpace)
	require.Equal(t, "/spaces-in-section/", sectionWithSpace.RelPermalink())

	th.assertFileContent("public/l1/l2/page/2/index.html", "L1/l2-IsActive: true", "PAG|T2_3|true")

}

func TestNextInSectionNested(t *testing.T) {
	t.Parallel()

	pageContent := `---
title: "The Page"
weight: %d
---
Some content.
`
	createPageContent := func(weight int) string {
		return fmt.Sprintf(pageContent, weight)
	}

	b := newTestSitesBuilder(t)
	b.WithSimpleConfigFile()
	b.WithTemplates("_default/single.html", `
Prev: {{ with .PrevInSection }}{{ .RelPermalink }}{{ end }}|
Next: {{ with .NextInSection }}{{ .RelPermalink }}{{ end }}|
`)

	b.WithContent("blog/page1.md", createPageContent(1))
	b.WithContent("blog/page2.md", createPageContent(2))
	b.WithContent("blog/cool/_index.md", createPageContent(1))
	b.WithContent("blog/cool/cool1.md", createPageContent(1))
	b.WithContent("blog/cool/cool2.md", createPageContent(2))
	b.WithContent("root1.md", createPageContent(1))
	b.WithContent("root2.md", createPageContent(2))

	b.Build(BuildCfg{})

	b.AssertFileContent("public/root1/index.html",
		"Prev: /root2/|", "Next: |")
	b.AssertFileContent("public/root2/index.html",
		"Prev: |", "Next: /root1/|")
	b.AssertFileContent("public/blog/page1/index.html",
		"Prev: /blog/page2/|", "Next: |")
	b.AssertFileContent("public/blog/page2/index.html",
		"Prev: |", "Next: /blog/page1/|")
	b.AssertFileContent("public/blog/cool/cool1/index.html",
		"Prev: /blog/cool/cool2/|", "Next: |")
	b.AssertFileContent("public/blog/cool/cool2/index.html",
		"Prev: |", "Next: /blog/cool/cool1/|")

}

func TestSectionsMultilingualBranchBundle(t *testing.T) {
	config := `
baseURL = "https://example.com"
defaultContentLanguage = "en"
defaultContentLanguageInSubdir = true

[Languages]
[Languages.en]
weight = 10
contentDir = "content/en"
[Languages.nn]
weight = 20
contentDir = "content/nn"
[Languages.sv]
weight = 20
contentDir = "content/sv"

`

	const pageContent = `---
title: %q
---
`
	createPage := func(s string) string {
		return fmt.Sprintf(pageContent, s)
	}

	b := newTestSitesBuilder(t).WithConfigFile("toml", config)

	b.WithTemplates("index.html", `{{ range .Site.Pages }}
{{ .Kind }}|{{ .Path }}|{{ with .CurrentSection }}CurrentSection: {{ .Path }}{{ end }}{{ end }}
`)

	b.WithContent("en/sect1/sect2/_index.md", createPage("en: Sect 2"))
	b.WithContent("en/sect1/sect2/page.md", createPage("en: Page"))
	b.WithContent("en/sect1/sect2/data.json", "mydata")
	b.WithContent("nn/sect1/sect2/page.md", createPage("nn: Page"))
	b.WithContent("nn/sect1/sect2/data.json", "my nn data")

	b.Build(BuildCfg{})

	b.AssertFileContent("public/en/index.html", "section|sect1/sect2/_index.md|CurrentSection: sect1/sect2/_index.md")
	b.AssertFileContent("public/nn/index.html", "page|sect1/sect2/page.md|CurrentSection: sect1")

}
