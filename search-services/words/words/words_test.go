package words

import (
	"strings"
	"testing"

	"slices"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNorm_SimpleWords(t *testing.T) {
	result := Norm("hello world")
	require.NotEmpty(t, result)
	assert.True(t, slices.Contains(result, "hello"))
	assert.True(t, slices.Contains(result, "world"))
}

func TestNorm_RemovesStopWords(t *testing.T) {
	result := Norm("the quick brown fox")
	assert.False(t, slices.Contains(result, "the"))
	assert.True(t, slices.Contains(result, "quick"))
	assert.True(t, slices.Contains(result, "brown"))
	assert.True(t, slices.Contains(result, "fox"))
}

func TestNorm_Stemming(t *testing.T) {
	result := Norm("running runs runner")
	assert.True(t, slices.Contains(result, "run"))
}

func TestNorm_CaseInsensitive(t *testing.T) {
	result1 := Norm("HELLO WORLD")
	result2 := Norm("hello world")
	assert.ElementsMatch(t, result1, result2)
}

func TestNorm_RemovesSpecialCharacters(t *testing.T) {
	result := Norm("hello-world! testing@123")
	require.NotEmpty(t, result)
	assert.True(t, slices.Contains(result, "hello"))
	assert.True(t, slices.Contains(result, "world"))
	assert.True(t, slices.Contains(result, "test"))
	assert.True(t, slices.Contains(result, "123"))
}

func TestNorm_EmptyString(t *testing.T) {
	result := Norm("")
	assert.Empty(t, result)
}

func TestNorm_OnlyStopWords(t *testing.T) {
	result := Norm("the a an")
	assert.Empty(t, result)
}

func TestNorm_WithNumbers(t *testing.T) {
	result := Norm("test abc numbers 123")
	require.NotEmpty(t, result)
	assert.True(t, slices.Contains(result, "test"))
	assert.True(t, slices.Contains(result, "abc"))
	assert.True(t, slices.Contains(result, "number"))
	assert.True(t, slices.Contains(result, "123"))
}

func TestNorm_SingleWord(t *testing.T) {
	result := Norm("winter")
	require.Len(t, result, 1)
	assert.Equal(t, "winter", result[0])
}

func TestNorm_ReturnsUnique(t *testing.T) {
	result := Norm("snow snow snow")
	assert.Len(t, result, 1)
	assert.Equal(t, "snow", result[0])
}

func TestNorm_ComplexPhrase(t *testing.T) {
	result := Norm("The quick brown fox jumps over the lazy dog")
	require.NotEmpty(t, result)
	assert.False(t, slices.Contains(result, "the"))
	assert.False(t, slices.Contains(result, "over"))
	assert.True(t, slices.Contains(result, "quick"))
	assert.True(t, slices.Contains(result, "brown"))
	assert.True(t, slices.Contains(result, "fox"))
	assert.True(t, slices.Contains(result, "jump"))
	assert.True(t, slices.Contains(result, "lazi"))
	assert.True(t, slices.Contains(result, "dog"))
}

func TestNorm_StopWordsWithNoise(t *testing.T) {
	result := Norm("the, a, an! 123")
	assert.Equal(t, []string{"123"}, result)
}

func TestNorm_LongString(t *testing.T) {
	longPhrase := strings.Repeat("happy Christmas ", 1000)
	result := Norm(longPhrase)
	assert.Contains(t, result, "happi")
	assert.Contains(t, result, "christma")
}

func TestNorm_PhraseWithAndOrThe(t *testing.T) {
	result := Norm("Happy Christmas and the New Year")
	assert.ElementsMatch(t, []string{"happi", "christma", "new", "year"}, result)
	assert.NotContains(t, result, "the")
	assert.NotContains(t, result, "and")
	assert.Len(t, result, 4)
}
