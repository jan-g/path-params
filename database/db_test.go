package database

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestSplitPathOnNull(t *testing.T) {
	parts := splitPath("")
	require.Equal(t, []prefixPath{{"", ""}}, parts)
}

func TestSplitPathOnNull2(t *testing.T) {
	parts := splitPath("/")
	require.Equal(t, []prefixPath{{"", ""}}, parts)
}

func TestSplitPath(t *testing.T) {
	parts := splitPath("/a/b/c")
	require.Equal(t, []prefixPath{
		{"", "a"},
		{"/a", "b"},
		{"/a/b", "c"},
		{"/a/b/c", ""},
	}, parts)
}
