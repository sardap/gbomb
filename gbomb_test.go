package gbomb

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

func createTestInvoker() *Invoker {
	invoker := CreateInvoker("https://www.giantbomb.com", "coolbeans")
	invoker.Limter = rate.NewLimiter(rate.Every(time.Duration(0)*time.Second), 1)

	return invoker
}

type PodcastFeedMock struct {
}

func (c *PodcastFeedMock) Do(req *http.Request) (*http.Response, error) {
	expectedURL, _ := url.Parse("https://www.giantbomb.com/feeds/podcast/?api_key=coolbeans")

	if req.URL.String() != expectedURL.String() {
		return nil, fmt.Errorf("invalid URL %s expected %s", req.URL, expectedURL)
	}

	file, _ := os.Open("test_data/bombcast_feed.xml")

	return &http.Response{
		Body:       file,
		StatusCode: 200,
		Status:     "200",
	}, nil
}

func TestRssChannel(t *testing.T) {
	invoker := createTestInvoker()
	invoker.client = &PodcastFeedMock{}
	feed, err := invoker.GetPodcasts("bombcast")
	if err != nil {
		t.Error(err)
	}

	if len(feed.Entries) != 810 {
		t.Errorf("invalid length read %d expected %d", len(feed.Entries), 810)
	}

	tme, _ := feed.Entries[0].GetPublishTime()
	expectedTme, _ := time.Parse(
		"2006-01-02 15:04:05 -0700 MST", "2021-02-09 14:52:00 +0000 PST",
	)
	if !tme.Equal(expectedTme) {
		t.Errorf(
			"didn't parse time correctly was %s expcted %s",
			tme.String(), expectedTme.String(),
		)
	}

	expctedDownload := "https://dts.podtrac.com/redirect.mp3/www.giantbomb.com/podcasts/download/3246/audio.mp3"
	if feed.Entries[0].link != expctedDownload {
		t.Errorf(
			"did not create download link correctly was %s expcted %s",
			feed.Entries[0].link, expctedDownload,
		)
	}
}

type GameMock struct {
}

func (g *GameMock) Do(req *http.Request) (*http.Response, error) {
	expectedURL, _ := url.Parse("https://www.giantbomb.com/api/game/3030-56733?api_key=coolbeans&format=json&offset=0")

	if req.URL.String() != expectedURL.String() {
		return nil, fmt.Errorf("invalid URL %s expected %s", req.URL, expectedURL)
	}

	file, _ := os.Open("test_data/gameRequest.json")

	return &http.Response{
		Body:       file,
		StatusCode: 200,
		Status:     "200",
	}, nil
}

func TestGetGame(t *testing.T) {
	invoker := createTestInvoker()
	invoker.client = &GameMock{}
	result, err := invoker.GetGame(context.Background(), "3030-56733")
	if err != nil {
		t.Error(err)
	}

	expcetedAPIURL := "https://www.giantbomb.com/api/game/3030-56733/"
	if result.APIDetailURL != expcetedAPIURL {
		t.Errorf(
			"invlaid API detital URL was %s expcted %s",
			result.APIDetailURL, expcetedAPIURL,
		)
	}

	if len(result.Videos) != 15 {
		t.Errorf(
			"did not parse videos correctly was %d expected %d",
			len(result.Videos), 15,
		)
	}

	if result.OriginalReleaseDate.String() != "2017-10-27" {
		t.Errorf(
			"did not prase OriginalReleaseDate correctly was %s expected %s",
			result.OriginalReleaseDate.String(), "2017-10-27",
		)
	}

	if len(result.Images) != 30 {
		t.Errorf("did not prase images correctly was %d expected %d",
			len(result.Images), 30,
		)
	}
}

type SearchGameMock struct {
	expectedURL string
}

func (g *SearchGameMock) Do(req *http.Request) (*http.Response, error) {
	if req.URL.String() != g.expectedURL {
		return nil, fmt.Errorf("invalid URL %s expected %s", req.URL, g.expectedURL)
	}

	file, _ := os.Open("test_data/gameSearch.json")

	return &http.Response{
		Body:       file,
		StatusCode: 200,
		Status:     "200",
	}, nil
}

func TestSearchGame(t *testing.T) {
	invoker := createTestInvoker()
	client := &SearchGameMock{
		expectedURL: "https://www.giantbomb.com/api/search?api_key=coolbeans&format=json&offset=0&query=Bangai-O&resources=game",
	}
	invoker.client = client
	result, err := invoker.SearchGame(context.Background(), "Bangai-O")
	if err != nil {
		t.Error(err)
	}

	if len(result.Results) != 10 {
		t.Errorf(
			"invalid number of results pulled %d expceted %d",
			len(result.Results), 10,
		)
	}

	if result.Results[0].Aliases != "Bakuretsu Muteki Bangai-O" {
		t.Errorf(
			"invalid aliases pulled %s expceted %s",
			result.Results[0].Aliases, "Bakuretsu Muteki Bangai-O",
		)
	}

	if result.Complete() {
		t.Errorf(
			"was complete early",
		)
	}

	client.expectedURL = "https://www.giantbomb.com/api/search?api_key=coolbeans&format=json&offset=10&query=Bangai-O&resources=game"
	err = invoker.Next(result)
	if err != nil {
		t.Error(errors.Wrapf(err, "next"))
	}

	client.expectedURL = "https://www.giantbomb.com/api/search?api_key=coolbeans&format=json&offset=10&query=Bangai-O&resources=game"
	err = invoker.Previous(result)
	if err == nil {
		t.Error(errors.Wrapf(err, "Prevoius"))
	}
}
