//+build faker

package usecase

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorable/github"
	githubModels "github.com/monitoror/monitoror/monitorable/github/models"
	"github.com/monitoror/monitoror/pkg/monitoror/builder"
	"github.com/monitoror/monitoror/pkg/monitoror/faker"
	"github.com/monitoror/monitoror/pkg/monitoror/utils/git"
	"github.com/monitoror/monitoror/pkg/monitoror/utils/nonempty"
)

type (
	githubUsecase struct {
		timeRefByProject map[string]time.Time
	}
)

var availableBuildStatus = faker.Statuses{
	{models.SuccessStatus, time.Second * 30},
	{models.FailedStatus, time.Second * 30},
	{models.CanceledStatus, time.Second * 20},
	{models.RunningStatus, time.Second * 60},
	{models.QueuedStatus, time.Second * 30},
	{models.WarningStatus, time.Second * 20},
	{models.DisabledStatus, time.Second * 20},
	{models.ActionRequiredStatus, time.Second * 20},
}

func NewGithubUsecase() github.Usecase {
	return &githubUsecase{make(map[string]time.Time)}
}

func (gu *githubUsecase) Issues(params *githubModels.IssuesParams) (*models.Tile, error) {
	tile := models.NewTile(github.GithubIssuesTileType)
	tile.Label = params.Query

	tile.Status = models.SuccessStatus

	if len(params.Values) != 0 {
		tile.Values = params.Values
	} else {
		tile.Values = []float64{42}
	}

	return tile, nil
}

func (gu *githubUsecase) Checks(params *githubModels.ChecksParams) (tile *models.Tile, err error) {
	tile = models.NewTile(github.GithubChecksTileType)
	tile.Label = fmt.Sprintf("%s\n%s", params.Repository, git.HumanizeBranch(params.Ref))

	tile.Status = nonempty.Struct(params.Status, gu.computeStatus(params)).(models.TileStatus)

	if tile.Status == models.DisabledStatus {
		return
	}

	if tile.Status == models.WarningStatus {
		// Warning can be Unstable Build
		if rand.Intn(2) == 0 {
			tile.Message = "random error message"
			return
		}
	}

	tile.PreviousStatus = nonempty.Struct(params.PreviousStatus, models.SuccessStatus).(models.TileStatus)

	// Author
	if tile.Status == models.FailedStatus {
		tile.Author = &models.Author{}
		tile.Author.Name = nonempty.String(params.AuthorName, "Faker")
		tile.Author.AvatarURL = nonempty.String(params.AuthorAvatarURL, "https://www.gravatar.com/avatar/00000000000000000000000000000000")
	}

	// Duration / EstimatedDuration
	if tile.Status == models.RunningStatus {
		estimatedDuration := nonempty.Duration(time.Duration(params.EstimatedDuration), time.Second*300)
		tile.Duration = pointer.ToInt64(nonempty.Int64(params.Duration, int64(gu.computeDuration(params, estimatedDuration).Seconds())))

		if tile.PreviousStatus != models.UnknownStatus {
			tile.EstimatedDuration = pointer.ToInt64(int64(estimatedDuration.Seconds()))
		} else {
			tile.EstimatedDuration = pointer.ToInt64(0)
		}
	}

	// StartedAt / FinishedAt
	if tile.Duration == nil {
		tile.StartedAt = pointer.ToTime(nonempty.Time(params.StartedAt, time.Now().Add(-time.Minute*10)))
	} else {
		tile.StartedAt = pointer.ToTime(nonempty.Time(params.StartedAt, time.Now().Add(-time.Second*time.Duration(*tile.Duration))))
	}

	if tile.Status != models.QueuedStatus && tile.Status != models.RunningStatus {
		tile.FinishedAt = pointer.ToTime(nonempty.Time(params.FinishedAt, tile.StartedAt.Add(time.Minute*5)))
	}

	return tile, nil
}

func (gu *githubUsecase) ListDynamicTile(params interface{}) ([]builder.Result, error) {
	panic("unimplemented")
}

func (gu *githubUsecase) computeStatus(params *githubModels.ChecksParams) models.TileStatus {
	projectID := fmt.Sprintf("%s-%s-%s", params.Owner, params.Repository, params.Ref)
	value, ok := gu.timeRefByProject[projectID]
	if !ok {
		gu.timeRefByProject[projectID] = faker.GetRefTime()
	}

	return faker.ComputeStatus(value, availableBuildStatus)
}

func (gu *githubUsecase) computeDuration(params *githubModels.ChecksParams, duration time.Duration) time.Duration {
	projectID := fmt.Sprintf("%s-%s-%s", params.Owner, params.Repository, params.Ref)
	value, ok := gu.timeRefByProject[projectID]
	if !ok {
		gu.timeRefByProject[projectID] = faker.GetRefTime()
	}

	return faker.ComputeDuration(value, duration)
}