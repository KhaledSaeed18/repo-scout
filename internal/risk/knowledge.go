package risk

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Owner is one author's share of the code, counted over the files where they
// made the most commits.
type Owner struct {
	Author       string    `json:"author"`
	Email        string    `json:"email"`
	Files        int       `json:"files"`
	Lines        int       `json:"lines"`
	Active       bool      `json:"active"`
	LastCommitAt time.Time `json:"lastCommitAt"`
}

// FolderKnowledge is how the code of one folder is spread across people.
type FolderKnowledge struct {
	Folder string `json:"folder"`
	Files  int    `json:"files"`
	Lines  int    `json:"lines"`
	Owners int    `json:"owners"`
	// TopOwner owns the most lines here; TopOwnerShare is that share (0..1).
	TopOwner      string  `json:"topOwner"`
	TopOwnerShare float64 `json:"topOwnerShare"`
	// InactiveShare is the share of lines owned by inactive authors.
	InactiveShare float64 `json:"inactiveShare"`
}

// AtRiskFile is a file whose main author has stopped committing.
type AtRiskFile struct {
	Path         string    `json:"path"`
	Lines        int       `json:"lines"`
	Owner        string    `json:"owner"`
	OwnerShare   float64   `json:"ownerShare"`
	LastCommitAt time.Time `json:"ownerLastCommitAt"`
}

// KnowledgeReport describes how concentrated knowledge of the code is.
type KnowledgeReport struct {
	// BusFactor is the fewest authors whose departure would leave more than
	// half of the code without its main author.
	BusFactor     int               `json:"busFactor"`
	Files         int               `json:"files"`
	Lines         int               `json:"lines"`
	InactiveLines int               `json:"inactiveLines"`
	Owners        []Owner           `json:"owners"`
	Folders       []FolderKnowledge `json:"folders"`
	AtRisk        []AtRiskFile      `json:"atRisk"`
	// ActiveSince is the cutoff: authors with a commit after it are active.
	// It counts back from the latest commit, not from today.
	ActiveSince *time.Time `json:"activeSince"`
}

// ownedFile is one file with its main author, as loaded from the database.
type ownedFile struct {
	Path      string
	Author    string
	Email     string
	Share     float64
	LinesCode int
}

// key identifies an author the same way the contributor rollup does.
func key(email, name string) string {
	if email != "" {
		return email
	}
	return name
}

// Knowledge reports how the code's main authorship is spread. Authors are
// inactive when they have not committed in inactiveMonths before the latest
// commit; folders are grouped depth levels deep.
func Knowledge(db *gorm.DB, repoID uint, depth, inactiveMonths int) (KnowledgeReport, error) {
	latest, err := latestCommit(db, repoID)
	if err != nil {
		return KnowledgeReport{}, err
	}
	if latest == nil {
		return buildKnowledge(nil, nil, time.Time{}, depth, 0), nil
	}
	var files []ownedFile
	err = db.Raw(`SELECT o.path, o.author, o.email, o.share, f.lines_code
		FROM file_ownerships o
		JOIN files f ON f.repo_id = o.repo_id AND f.path = o.path
		WHERE o.repo_id = ?`, repoID).Scan(&files).Error
	if err != nil {
		return KnowledgeReport{}, fmt.Errorf("load ownership: %w", err)
	}
	var people []struct {
		Name         string
		Email        string
		LastCommitAt time.Time
	}
	if err := db.Table("contributors").Select("name", "email", "last_commit_at").
		Where("repo_id = ?", repoID).Scan(&people).Error; err != nil {
		return KnowledgeReport{}, fmt.Errorf("load contributors: %w", err)
	}
	lastSeen := make(map[string]time.Time, len(people))
	for _, p := range people {
		lastSeen[key(p.Email, p.Name)] = p.LastCommitAt
	}
	cutoff := latest.AddDate(0, -inactiveMonths, 0)
	return buildKnowledge(files, lastSeen, cutoff, depth, 25), nil
}

// buildKnowledge does the rollups. Code is weighted by lines of code, with
// every file counting at least one line so config and docs still register.
func buildKnowledge(files []ownedFile, lastSeen map[string]time.Time, cutoff time.Time, depth, atRiskLimit int) KnowledgeReport {
	rep := KnowledgeReport{Owners: []Owner{}, Folders: []FolderKnowledge{}, AtRisk: []AtRiskFile{}}
	if !cutoff.IsZero() {
		rep.ActiveSince = &cutoff
	}
	active := func(k string) bool { return !lastSeen[k].Before(cutoff) }

	owners := map[string]*Owner{}
	type folderAcc struct {
		FolderKnowledge
		byOwner map[string]int
	}
	folders := map[string]*folderAcc{}
	for _, f := range files {
		lines := max(f.LinesCode, 1)
		k := key(f.Email, f.Author)
		o := owners[k]
		if o == nil {
			o = &Owner{Author: f.Author, Email: f.Email, Active: active(k), LastCommitAt: lastSeen[k]}
			owners[k] = o
		}
		o.Files++
		o.Lines += lines
		rep.Files++
		rep.Lines += lines
		if !o.Active {
			rep.InactiveLines += lines
			rep.AtRisk = append(rep.AtRisk, AtRiskFile{
				Path: f.Path, Lines: f.LinesCode, Owner: f.Author, OwnerShare: f.Share, LastCommitAt: lastSeen[k],
			})
		}

		name := folderAt(f.Path, depth)
		fa := folders[name]
		if fa == nil {
			fa = &folderAcc{FolderKnowledge: FolderKnowledge{Folder: name}, byOwner: map[string]int{}}
			folders[name] = fa
		}
		fa.Files++
		fa.Lines += lines
		fa.byOwner[k] += lines
		if !o.Active {
			fa.InactiveShare += float64(lines)
		}
	}

	for _, o := range owners {
		rep.Owners = append(rep.Owners, *o)
	}
	sort.Slice(rep.Owners, func(i, j int) bool {
		a, b := rep.Owners[i], rep.Owners[j]
		if a.Lines != b.Lines {
			return a.Lines > b.Lines
		}
		return a.Author < b.Author
	})
	rep.BusFactor = busFactor(rep.Owners, rep.Lines)

	for _, fa := range folders {
		f := fa.FolderKnowledge
		f.Owners = len(fa.byOwner)
		best := ""
		for k, lines := range fa.byOwner {
			if lines > fa.byOwner[best] || (lines == fa.byOwner[best] && k < best) {
				best = k
			}
		}
		f.TopOwner = owners[best].Author
		f.TopOwnerShare = float64(fa.byOwner[best]) / float64(f.Lines)
		f.InactiveShare /= float64(f.Lines)
		rep.Folders = append(rep.Folders, f)
	}
	sort.Slice(rep.Folders, func(i, j int) bool {
		a, b := rep.Folders[i], rep.Folders[j]
		ai, bi := a.InactiveShare*float64(a.Lines), b.InactiveShare*float64(b.Lines)
		if ai != bi {
			return ai > bi
		}
		if a.Lines != b.Lines {
			return a.Lines > b.Lines
		}
		return a.Folder < b.Folder
	})

	sort.Slice(rep.AtRisk, func(i, j int) bool {
		if rep.AtRisk[i].Lines != rep.AtRisk[j].Lines {
			return rep.AtRisk[i].Lines > rep.AtRisk[j].Lines
		}
		return rep.AtRisk[i].Path < rep.AtRisk[j].Path
	})
	if len(rep.AtRisk) > atRiskLimit {
		rep.AtRisk = rep.AtRisk[:atRiskLimit]
	}
	return rep
}

// busFactor removes the biggest owners one by one until more than half of the
// lines have lost their main author, and returns how many it took. owners
// must be sorted by lines, largest first.
func busFactor(owners []Owner, total int) int {
	orphaned := 0
	for i, o := range owners {
		orphaned += o.Lines
		if orphaned*2 > total {
			return i + 1
		}
	}
	return len(owners)
}

// folderAt returns the folder holding path, cut to at most depth levels.
// Files at the top level belong to "".
func folderAt(path string, depth int) string {
	parts := strings.Split(path, "/")
	parts = parts[:len(parts)-1]
	if len(parts) > depth {
		parts = parts[:depth]
	}
	return strings.Join(parts, "/")
}
