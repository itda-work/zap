package issue

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ConflictType represents the type of number conflict.
type ConflictType string

const (
	// ConflictDuplicateFilename means multiple files have the same number prefix.
	ConflictDuplicateFilename ConflictType = "duplicate_filename"
	// ConflictDuplicateFrontmatter means multiple files have the same frontmatter number.
	ConflictDuplicateFrontmatter ConflictType = "duplicate_frontmatter"
	// ConflictMismatch means filename number differs from frontmatter number.
	ConflictMismatch ConflictType = "mismatch"
)

// FileInfo holds information about an issue file for conflict detection.
type FileInfo struct {
	FilePath        string
	FileName        string
	FilenameNumber  int       // Number extracted from filename (NNN-slug.md)
	FrontmatterNum  int       // Number from frontmatter
	CreatedAt       time.Time // From frontmatter or git
	Issue           *Issue    // Parsed issue (nil if parse failed)
	ParseError      string    // Error message if parse failed
	GitCreatedAt    *time.Time // From git log (nil if not in git)
}

// Conflict represents a detected number conflict.
type Conflict struct {
	Type        ConflictType
	Number      int         // The conflicting number
	Files       []*FileInfo // Files involved in this conflict
	ToRenumber  *FileInfo   // File that should be renumbered (later created)
	NewNumber   int         // New number to assign
	Description string      // Human-readable description
}

// ConflictDetector detects number conflicts in issue files.
type ConflictDetector struct {
	baseDir string
	gitRoot string // Git repository root (empty if not in git)
}

// NewConflictDetector creates a new conflict detector.
func NewConflictDetector(baseDir string) *ConflictDetector {
	cd := &ConflictDetector{baseDir: baseDir}
	cd.gitRoot = cd.findGitRoot()
	return cd
}

// findGitRoot finds the git repository root.
func (cd *ConflictDetector) findGitRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cd.baseDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// DetectConflicts scans the issues directory and detects all conflicts.
func (cd *ConflictDetector) DetectConflicts() ([]*Conflict, error) {
	files, err := cd.loadAllFiles()
	if err != nil {
		return nil, err
	}

	var conflicts []*Conflict

	// Detect duplicate filename numbers
	filenameConflicts := cd.detectDuplicateFilenames(files)
	conflicts = append(conflicts, filenameConflicts...)

	// Detect duplicate frontmatter numbers
	frontmatterConflicts := cd.detectDuplicateFrontmatters(files)
	conflicts = append(conflicts, frontmatterConflicts...)

	// Detect filename-frontmatter mismatches
	mismatchConflicts := cd.detectMismatches(files)
	conflicts = append(conflicts, mismatchConflicts...)

	// For each conflict, determine which file to renumber and assign new numbers
	cd.resolveConflicts(conflicts, files)

	return conflicts, nil
}

// loadAllFiles loads information about all .md files in the issues directory.
func (cd *ConflictDetector) loadAllFiles() ([]*FileInfo, error) {
	entries, err := os.ReadDir(cd.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read issues directory: %w", err)
	}

	var files []*FileInfo
	filenamePattern := regexp.MustCompile(`^(\d+)-`)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(cd.baseDir, entry.Name())
		fi := &FileInfo{
			FilePath: filePath,
			FileName: entry.Name(),
		}

		// Extract number from filename
		if matches := filenamePattern.FindStringSubmatch(entry.Name()); len(matches) > 1 {
			if num, err := strconv.Atoi(matches[1]); err == nil {
				fi.FilenameNumber = num
			}
		}

		// Try to parse the issue
		issue, err := Parse(filePath)
		if err != nil {
			fi.ParseError = err.Error()
		} else {
			fi.Issue = issue
			fi.FrontmatterNum = issue.Number
			fi.CreatedAt = issue.CreatedAt
		}

		// Get git creation time
		fi.GitCreatedAt = cd.getGitCreatedAt(filePath)

		files = append(files, fi)
	}

	return files, nil
}

// getGitCreatedAt returns the first commit time for a file.
func (cd *ConflictDetector) getGitCreatedAt(filePath string) *time.Time {
	if cd.gitRoot == "" {
		return nil
	}

	// Get the first commit that added this file
	cmd := exec.Command("git", "log", "--diff-filter=A", "--follow", "--format=%aI", "--", filePath)
	cmd.Dir = cd.gitRoot
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil
	}

	// Parse the ISO 8601 date
	t, err := time.Parse(time.RFC3339, lines[len(lines)-1]) // Last line is the first commit
	if err != nil {
		return nil
	}

	return &t
}

// GetEffectiveCreatedAt returns the creation time to use for sorting.
// Priority: git log > created_at from frontmatter
func (fi *FileInfo) GetEffectiveCreatedAt() time.Time {
	if fi.GitCreatedAt != nil {
		return *fi.GitCreatedAt
	}
	if !fi.CreatedAt.IsZero() {
		return fi.CreatedAt
	}
	// Fallback to current time (treat as newest)
	return time.Now()
}

// detectDuplicateFilenames finds files with the same filename number prefix.
func (cd *ConflictDetector) detectDuplicateFilenames(files []*FileInfo) []*Conflict {
	byNumber := make(map[int][]*FileInfo)

	for _, fi := range files {
		if fi.FilenameNumber > 0 {
			byNumber[fi.FilenameNumber] = append(byNumber[fi.FilenameNumber], fi)
		}
	}

	var conflicts []*Conflict
	for num, fis := range byNumber {
		if len(fis) > 1 {
			conflicts = append(conflicts, &Conflict{
				Type:   ConflictDuplicateFilename,
				Number: num,
				Files:  fis,
				Description: fmt.Sprintf("Multiple files have filename number %03d: %s",
					num, fileNames(fis)),
			})
		}
	}

	return conflicts
}

// detectDuplicateFrontmatters finds files with the same frontmatter number.
func (cd *ConflictDetector) detectDuplicateFrontmatters(files []*FileInfo) []*Conflict {
	byNumber := make(map[int][]*FileInfo)

	for _, fi := range files {
		if fi.Issue != nil && fi.FrontmatterNum > 0 {
			byNumber[fi.FrontmatterNum] = append(byNumber[fi.FrontmatterNum], fi)
		}
	}

	var conflicts []*Conflict
	for num, fis := range byNumber {
		if len(fis) > 1 {
			// Skip if this is also a filename conflict (to avoid double reporting)
			allSameFilename := true
			for _, fi := range fis {
				if fi.FilenameNumber != num {
					allSameFilename = false
					break
				}
			}
			if allSameFilename {
				continue // Already reported as filename conflict
			}

			conflicts = append(conflicts, &Conflict{
				Type:   ConflictDuplicateFrontmatter,
				Number: num,
				Files:  fis,
				Description: fmt.Sprintf("Multiple files have frontmatter number %d: %s",
					num, fileNames(fis)),
			})
		}
	}

	return conflicts
}

// detectMismatches finds files where filename number differs from frontmatter number.
func (cd *ConflictDetector) detectMismatches(files []*FileInfo) []*Conflict {
	var conflicts []*Conflict

	for _, fi := range files {
		if fi.Issue != nil && fi.FilenameNumber > 0 && fi.FrontmatterNum > 0 {
			if fi.FilenameNumber != fi.FrontmatterNum {
				conflicts = append(conflicts, &Conflict{
					Type:   ConflictMismatch,
					Number: fi.FilenameNumber,
					Files:  []*FileInfo{fi},
					Description: fmt.Sprintf("File %s has filename number %03d but frontmatter number %d",
						fi.FileName, fi.FilenameNumber, fi.FrontmatterNum),
				})
			}
		}
	}

	return conflicts
}

// resolveConflicts determines which file to renumber and assigns new numbers.
func (cd *ConflictDetector) resolveConflicts(conflicts []*Conflict, allFiles []*FileInfo) {
	// Find the maximum number currently in use
	maxNumber := 0
	for _, fi := range allFiles {
		if fi.FilenameNumber > maxNumber {
			maxNumber = fi.FilenameNumber
		}
		if fi.FrontmatterNum > maxNumber {
			maxNumber = fi.FrontmatterNum
		}
	}

	nextNumber := maxNumber + 1

	for _, conflict := range conflicts {
		switch conflict.Type {
		case ConflictDuplicateFilename, ConflictDuplicateFrontmatter:
			// Find the later-created file to renumber
			conflict.ToRenumber = findLaterCreated(conflict.Files)
			conflict.NewNumber = nextNumber
			nextNumber++

		case ConflictMismatch:
			// For mismatch, we update frontmatter to match filename
			conflict.ToRenumber = conflict.Files[0]
			conflict.NewNumber = conflict.Files[0].FilenameNumber
		}
	}
}

// findLaterCreated returns the file that was created later.
func findLaterCreated(files []*FileInfo) *FileInfo {
	if len(files) == 0 {
		return nil
	}

	latest := files[0]
	latestTime := latest.GetEffectiveCreatedAt()

	for _, fi := range files[1:] {
		t := fi.GetEffectiveCreatedAt()
		if t.After(latestTime) {
			latest = fi
			latestTime = t
		}
	}

	return latest
}

// fileNames returns a comma-separated list of filenames.
func fileNames(files []*FileInfo) string {
	names := make([]string, len(files))
	for i, fi := range files {
		names[i] = fi.FileName
	}
	return strings.Join(names, ", ")
}

// GetAllIssueContents returns all issue file contents for AI context.
func (cd *ConflictDetector) GetAllIssueContents() (map[string]string, error) {
	entries, err := os.ReadDir(cd.baseDir)
	if err != nil {
		return nil, err
	}

	contents := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(cd.baseDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		contents[entry.Name()] = string(data)
	}

	return contents, nil
}
