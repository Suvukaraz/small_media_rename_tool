package renamer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Regex patterns to strip SRT formatting for content comparison
var (
	// Matches SRT sequence numbers (standalone lines with just a number)
	srtSequenceNum = regexp.MustCompile(`(?m)^\d+\s*$`)
	// Matches SRT/VTT timestamps like "00:01:23,456 --> 00:01:25,789"
	srtTimestamp = regexp.MustCompile(`(?m)^\d{1,2}:\d{2}:\d{2}[,.]\d{1,3}\s*-->\s*\d{1,2}:\d{2}:\d{2}[,.]\d{1,3}.*$`)
	// Matches formatting tags: <b>, <i>, {\an8}, {\\pos(...)}, [Music], etc.
	srtFormatTags = regexp.MustCompile(`<[^>]+>|\{[^}]*\}|\[[^\]]*\]`)
	// Matches lines that are only whitespace
	blankLines = regexp.MustCompile(`(?m)^\s*$`)
)

// NormalizeSubtitleContent strips all formatting, timestamps, and sequence numbers
// from subtitle content, leaving only the pure dialog text (lowercased).
// This allows content-based comparison regardless of formatting differences.
func NormalizeSubtitleContent(content string) string {
	// Remove BOM
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")

	// Remove timestamps
	content = srtTimestamp.ReplaceAllString(content, "")
	// Remove sequence numbers
	content = srtSequenceNum.ReplaceAllString(content, "")
	// Remove formatting tags
	content = srtFormatTags.ReplaceAllString(content, "")
	// Remove WEBVTT header if present
	content = strings.Replace(content, "WEBVTT", "", 1)

	// Normalize whitespace: collapse multiple spaces/newlines into single space
	content = blankLines.ReplaceAllString(content, "")
	fields := strings.Fields(content)
	normalized := strings.Join(fields, " ")

	// Lowercase for comparison
	return strings.ToLower(strings.TrimSpace(normalized))
}

// ContentHash computes a SHA-256 hash of normalized subtitle content.
func ContentHash(normalizedContent string) string {
	h := sha256.Sum256([]byte(normalizedContent))
	return fmt.Sprintf("%x", h)
}

// SubtitleWithHash extends SubtitleInfo with content hash for deduplication.
type SubtitleWithHash struct {
	SubtitleInfo
	ContentHash string
	FileSize    int64
}

// DeduplicateSubtitles takes a list of valid (DE/EN) subtitles and returns
// two slices: the subtitles to keep and the subtitles to delete as duplicates.
// Duplicates are identified by having identical normalized content hashes
// within the same language group. The largest file per group is kept.
func DeduplicateSubtitles(subs []SubtitleInfo) (keep []SubtitleInfo, duplicates []DuplicateInfo) {
	if len(subs) <= 1 {
		return subs, nil
	}

	// Read and hash each subtitle
	var hashedSubs []SubtitleWithHash
	for _, sub := range subs {
		content, err := readSubtitleSample(sub.OriginalPath, 15*1024*1024)
		if err != nil {
			// Can't read file -> keep it (don't risk deleting)
			hashedSubs = append(hashedSubs, SubtitleWithHash{
				SubtitleInfo: sub,
				ContentHash:  fmt.Sprintf("unreadable_%s", sub.OriginalPath),
				FileSize:     0,
			})
			continue
		}

		normalized := NormalizeSubtitleContent(content)
		hash := ContentHash(normalized)

		var fileSize int64
		if info, err := os.Stat(sub.OriginalPath); err == nil {
			fileSize = info.Size()
		}

		hashedSubs = append(hashedSubs, SubtitleWithHash{
			SubtitleInfo: sub,
			ContentHash:  hash,
			FileSize:     fileSize,
		})
	}

	// Group by language + content hash
	// Key: "de:abc123" or "en:abc123"
	type groupKey struct {
		lang string
		hash string
	}

	groups := make(map[groupKey][]int) // maps to indices in hashedSubs
	for i, hs := range hashedSubs {
		key := groupKey{lang: hs.LangCode, hash: hs.ContentHash}
		groups[key] = append(groups[key], i)
	}

	// For each group with >1 member, keep the largest file, mark others as duplicates
	keptIndices := make(map[int]bool)
	dupIndices := make(map[int]int) // maps duplicate index -> kept index

	for _, indices := range groups {
		if len(indices) <= 1 {
			// Single file in group -> keep
			for _, idx := range indices {
				keptIndices[idx] = true
			}
			continue
		}

		// Find the largest file in the group
		bestIdx := indices[0]
		for _, idx := range indices[1:] {
			if hashedSubs[idx].FileSize > hashedSubs[bestIdx].FileSize {
				bestIdx = idx
			}
		}

		keptIndices[bestIdx] = true
		for _, idx := range indices {
			if idx != bestIdx {
				dupIndices[idx] = bestIdx
			}
		}
	}

	// Build result slices
	for i, hs := range hashedSubs {
		if keptIndices[i] {
			keep = append(keep, hs.SubtitleInfo)
		} else if keptByIdx, isDup := dupIndices[i]; isDup {
			duplicates = append(duplicates, DuplicateInfo{
				Duplicate:    hs.SubtitleInfo,
				KeptOriginal: hashedSubs[keptByIdx].SubtitleInfo,
			})
		}
	}

	return keep, duplicates
}

// DuplicateInfo holds information about a detected duplicate subtitle.
type DuplicateInfo struct {
	Duplicate    SubtitleInfo // The file to be deleted
	KeptOriginal SubtitleInfo // The file that will be kept
}
