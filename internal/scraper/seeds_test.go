package scraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrimarySkillsRepoExists(t *testing.T) {
	assert.Equal(t, "asteroid-belt", PrimarySkillsRepo.Owner)
	assert.Equal(t, "skills", PrimarySkillsRepo.Repo)
	assert.Equal(t, 10, PrimarySkillsRepo.Priority)
	assert.Equal(t, "official", PrimarySkillsRepo.Type)
}

func TestAllSeedsIncludesPrimary(t *testing.T) {
	seeds := AllSeeds()

	found := false
	for _, seed := range seeds {
		if seed.Owner == "asteroid-belt" && seed.Repo == "skills" {
			found = true
			break
		}
	}

	assert.True(t, found, "Primary skills repo should be in AllSeeds()")
}

func TestOfficialSeedsIncludesPrimary(t *testing.T) {
	found := false
	for _, seed := range OfficialSeeds {
		if seed.Owner == "asteroid-belt" && seed.Repo == "skills" {
			found = true
			break
		}
	}

	assert.True(t, found, "Primary skills repo should be first in OfficialSeeds")
}

func TestPrimarySkillsRepoMatchesOfficialSeed(t *testing.T) {
	// Ensure PrimarySkillsRepo matches the entry in OfficialSeeds
	for _, seed := range OfficialSeeds {
		if seed.Owner == PrimarySkillsRepo.Owner && seed.Repo == PrimarySkillsRepo.Repo {
			assert.Equal(t, seed.Priority, PrimarySkillsRepo.Priority)
			assert.Equal(t, seed.Type, PrimarySkillsRepo.Type)
			return
		}
	}
	t.Fatal("PrimarySkillsRepo not found in OfficialSeeds")
}
