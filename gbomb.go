package gbomb

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

//Invoker Invoker
type Invoker struct {
	Endpoint string
	APIKey   string
	Limter   *rate.Limiter
	client   httpClient
}

func (i *Invoker) requestLimiter() {
	i.Limter.Wait(context.TODO())
}

func (i *Invoker) Get(pageable Pageable) ([]byte, error) {
	path, query := pageable.Path()

	url := fmt.Sprintf("%s/%s", i.Endpoint, path)

	req, err := http.NewRequest(
		"GET", url, nil,
	)
	if err != nil {
		return nil, err
	}

	offset := pageable.GetOffset()
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("api_key", i.APIKey)
	q.Add("format", "json")
	q.Add("offset", fmt.Sprintf("%d", offset))
	for k, v := range query {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	i.requestLimiter()
	res, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

//Next gets next page for pageable
func (i *Invoker) Next(page Pageable) error {
	if err := page.NextOffset(); err != nil {
		return err
	}

	body, err := i.Get(page)
	if err != nil {
		return err
	}

	page.Parse(body)

	return nil
}

//Next gets next page for pageable
func (i *Invoker) Previous(page Pageable) error {
	if err := page.PreviousOffset(); err != nil {
		return err
	}

	body, err := i.Get(page)
	if err != nil {
		return err
	}

	page.Parse(body)

	return nil
}

//CreateInvoker Creates a giant bomb invoker
func CreateInvoker(endpoint, key string) *Invoker {
	return &Invoker{
		Endpoint: endpoint, APIKey: key,
		Limter: rate.NewLimiter(rate.Every(time.Duration(31)*time.Second), 1),
		client: http.DefaultClient,
	}
}

//Pageable used for reuqests with many pages
type Pageable interface {
	NextOffset() error
	PreviousOffset() error
	GetOffset() int
	Complete() bool
	Path() (string, map[string]string)
	Parse(data []byte) error
}

//Date a giant bomb time
type Date struct {
	date string
}

//UnmarshalJSON custom json unmarshaler
func (d *Date) UnmarshalJSON(data []byte) error {
	//someone \" are being included
	d.date = string(data[1 : len(data)-1])
	return nil
}

//String returns date as string
func (d *Date) String() string {
	return d.date
}

//GetTime converts from giant bomb date to time.Time
func (d *Date) GetTime() time.Time {
	var layout string
	if len(d.date) == 19 {
		layout = "2006-01-02 15:04:05"
	} else if len(d.date) == 10 {
		layout = "2006-01-02"
	} else {
		layout = ""
	}

	result, _ := time.Parse(
		layout,
		string(d.date),
	)

	return result
}

//Image a giant bomb API Image
type Image struct {
	IconURL        string `json:"icon_url"`
	MediumURL      string `json:"medium_url"`
	ScreenURL      string `json:"screen_url"`
	ScreenLargeURL string `json:"screen_large_url"`
	SmallURL       string `json:"small_url"`
	SuperURL       string `json:"super_url"`
	ThumbURL       string `json:"thumb_url"`
	TinyURL        string `json:"tiny_url"`
	OriginalURL    string `json:"original_url"`
	ImageTags      string `json:"image_tags"`
}

//Tag tag
type Tag struct {
	APIDetailURL string `json:"api_detail_url"`
	Name         string `json:"name"`
}

//ImageTag a giant bomb API image tag
type ImageTag struct {
	Tag
	Total int `json:"total"`
}

//GameRatingTag GameRatingTag
type GameRatingTag struct {
	Tag
	ID int `json:"id"`
}

//CompleteTag CompleteTag
type CompleteTag struct {
	Tag
	ID            int    `json:"id"`
	SiteDetailURL string `json:"site_detail_url"`
}

//PlatformTag PlatformTag
type PlatformTag struct {
	CompleteTag
	Abbreviation string `json:"abbreviation"`
}

//VideoShow Giant bomb api VideoShow
type VideoShow struct {
	APIDetailURL  string `json:"api_detail_url"`
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Postion       int    `json:"postion"`
	SiteDetailURL string `json:"site_detail_url"`
	Image         Image  `json:"image"`
	Logo          Image  `json:"logo"`
}

//Association an Association
type Association struct {
	APIDetailURL  string `json:"api_detail_url"`
	SiteDetailURL string `json:"site_detail_url"`
	GUID          string `json:"guid"`
	ID            string `json:"id"`
	Name          string `json:"name"`
}

//VideoCategory a VideoCategory
type VideoCategory struct {
	APIDetailURL  string `json:"api_detail_url"`
	SiteDetailURL string `json:"site_detail_url"`
	ID            string `json:"id"`
	Name          string `json:"name"`
}

//VideoInfo a VideoInfo
type VideoInfo struct {
	DetailURL       string          `json:"api_detail_url"`
	SiteDetailURL   string          `json:"site_detail_url"`
	GUID            string          `json:"guid"`
	ID              string          `json:"id"`
	Associations    []Association   `json:"associations"`
	Deck            string          `json:"deck"`
	EmbedPlayer     string          `json:"embed_player"`
	LengthSeconds   int             `json:"length_seconds"`
	Name            string          `json:"name"`
	Premium         bool            `json:"premium"`
	PublishDate     Date            `json:"publish_date"`
	User            string          `json:"user"`
	Hosts           string          `json:"Hosts"`
	Crew            string          `json:"crew"`
	VideoType       string          `json:"video_type"`
	Show            VideoShow       `json:"video_show"`
	VideoCategories []VideoCategory `json:"video_categories"`
	SavedTime       string          `json:"saved_time"`
	YoutubeID       string          `json:"youtube_id"`
	LowURL          string          `json:"low_url"`
	HighURL         string          `json:"high_url"`
	HDURL           string          `json:"hd_url"`
	URL             string          `json:"url"`
}

//LengthDuration Returns the legnth as time.Duration
func (v *VideoInfo) LengthDuration() time.Duration {
	return time.Duration(v.LengthSeconds) * time.Second
}

//OnYoutube returns if video can be found on youtube
func (v *VideoInfo) OnYoutube() bool {
	return v.YoutubeID != ""
}

//GetHighestURL returns high URL if valid or low URL
func (v *VideoInfo) GetHighestURL() string {
	if v.HighURL != "" {
		return v.HighURL
	}

	return v.LowURL
}

//GetBestQuailtyURL returns the best Quailty video url
func (v *VideoInfo) GetBestQuailtyURL() string {
	if v.HDURL != "" {
		return v.HDURL
	}

	return v.GetHighestURL()
}

//Game a game
type Game struct {
	Aliases                   string          `json:"aliases"`
	APIDetailURL              string          `json:"api_detail_url"`
	SiteDetailURL             string          `json:"site_detail_url"`
	GUID                      string          `json:"guid"`
	ID                        int             `json:"id"`
	DateAdded                 Date            `json:"date_added"`
	DateLastUpdate            Date            `json:"date_last_updated"`
	Deck                      string          `json:"dec"`
	Description               string          `json:"description"`
	ExpectedReleaseDay        int             `json:"expected_release_day"`
	ExpectedReleaseMonth      int             `json:"expected_release_month"`
	ExpectedReleaseQuarter    int             `json:"expected_release_quarter"`
	ExpectedReleaseYear       int             `json:"expected_release_year"`
	Image                     Image           `json:"image"`
	ImageTags                 []ImageTag      `json:"image_tags"`
	Images                    []Image         `json:"images"`
	Name                      string          `json:"name"`
	NumberOfUserReviews       int             `json:"number_of_user_reviews"`
	OriginalGameRating        []GameRatingTag `json:"original_game_rating"`
	OriginalReleaseDate       Date            `json:"original_release_date"`
	Platforms                 []PlatformTag   `json:"platforms"`
	Videos                    []CompleteTag   `json:"videos"`
	Characters                []CompleteTag   `json:"characters"`
	Concepts                  []CompleteTag   `json:"concepts"`
	Developers                []CompleteTag   `json:"developers"`
	FirstAppearanceCharacters []CompleteTag   `json:"first_appearance_characters"`
	FirstAppearanceConcepts   []CompleteTag   `json:"first_appearance_concepts"`
	FirstAppearanceLocations  []CompleteTag   `json:"first_appearance_locations"`
	FirstAppearancePeople     []CompleteTag   `json:"first_appearance_people"`
	Franchises                []CompleteTag   `json:"franchises"`
	Genres                    []CompleteTag   `json:"genres"`
	KilledCharacters          []CompleteTag   `json:"killed_characters"`
	Locations                 []CompleteTag   `json:"locations"`
	Objects                   []CompleteTag   `json:"objects"`
	Persons                   []CompleteTag   `json:"people"`
	Publishers                []CompleteTag   `json:"publishers"`
	Releases                  []CompleteTag   `json:"releases"`
	DLCS                      []CompleteTag   `json:"dlcs"`
	Reviews                   []CompleteTag   `json:"reviews"`
	SimilarGames              []CompleteTag   `json:"similar_games"`
	Themes                    []CompleteTag   `json:"themes"`
}

//ResponsePage the page part of a response
type ResponsePage struct {
	Error       string `json:"error"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	PageResults int    `json:"number_of_page_results"`
	MaxResults  int    `json:"number_of_total_results"`
	StatusCode  int    `json:"status_code"`
}

//NextOffset returns the next offset
func (r *ResponsePage) NextOffset() error {
	if r.Offset > r.MaxResults {
		return fmt.Errorf("no more results")
	}

	next := r.MaxResults - r.Offset
	if next > r.Limit {
		next = r.Limit
	}

	r.Offset += next

	return nil

}

//NextOffset returns the next offset
func (r *ResponsePage) PreviousOffset() error {
	if r.Offset-1 < 0 {
		return fmt.Errorf("no more results")
	}

	r.Offset -= r.MaxResults
	if r.Offset < 0 {
		r.Offset = 0
	}

	return nil
}

//GetOffset returns the next offset
func (r *ResponsePage) GetOffset() int {
	return r.Offset
}

//Complete will return if there are no more results
func (r *ResponsePage) Complete() bool {
	return r.Offset >= r.MaxResults
}

//VideosResponse videos response from giant bomb API
type VideosResponse struct {
	ResponsePage
	Videos []VideoInfo `json:"results"`
}

//Path returns video path
func (v *VideosResponse) Path() (string, map[string]string) {
	return fmt.Sprintf("api/videos"), make(map[string]string)
}

//Parse parse
func (v *VideosResponse) Parse(data []byte) error {
	var tmp VideosResponse
	json.Unmarshal(data, &tmp)

	v.ResponsePage = tmp.ResponsePage
	v.Videos = tmp.Videos

	return nil
}

//GetVideos returns a base video
func (i *Invoker) GetVideos(ctx context.Context, offset int) (*VideosResponse, error) {
	result := &VideosResponse{}
	result.Offset = offset

	body, err := i.Get(result)
	if err != nil {
		return nil, err
	}

	result.Parse(body)

	return result, nil
}

//DownloadVideo downloads a given video
func (i *Invoker) DownloadVideo(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("api_key", i.APIKey)
	req.URL.RawQuery = q.Encode()

	i.requestLimiter()
	res, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}

type gameResponseInternal struct {
	ResponsePage
	Results   *Game `json:"results"`
	tagetGame string
}

//Path returns video path
func (g *gameResponseInternal) Path() (string, map[string]string) {
	return fmt.Sprintf("api/game/%s", g.tagetGame), make(map[string]string)
}

//Parse parse
func (g *gameResponseInternal) Parse(data []byte) error {
	var tmp gameResponseInternal
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	g.ResponsePage = tmp.ResponsePage
	g.Results = tmp.Results
	g.tagetGame = tmp.tagetGame

	return nil
}

//GetGame returns a given game
func (i *Invoker) GetGame(ctx context.Context, gameID string) (*Game, error) {
	result := &gameResponseInternal{}

	result.tagetGame = gameID

	body, err := i.Get(result)
	if err != nil {
		return nil, err
	}

	err = result.Parse(body)
	if err != nil {
		return nil, err
	}

	return result.Results, err
}

//GamesResponse game response
type GamesResponse struct {
	ResponsePage
	Results   []Game `json:"results"`
	tagetGame string
}

//Path returns path for game Serach
func (g *GamesResponse) Path() (string, map[string]string) {
	return "api/search", map[string]string{
		"query":     g.tagetGame,
		"resources": "game",
	}
}

//Parse parse
func (g *GamesResponse) Parse(data []byte) error {
	var tmp GamesResponse
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	g.ResponsePage = tmp.ResponsePage
	g.Results = tmp.Results

	return nil
}

//SearchGame search game
func (i *Invoker) SearchGame(ctx context.Context, name string) (*GamesResponse, error) {
	result := &GamesResponse{}

	result.tagetGame = name

	body, err := i.Get(result)
	if err != nil {
		return nil, err
	}

	err = result.Parse(body)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// #####################################################################

//RSSFeedEntry a giant bomb RSS feed entry
type RSSFeedEntry struct {
	Title      string `xml:"title"`
	PubDateStr string `xml:"pubDate"`
	GUID       string `xml:"guid"`
	//This is constructed not pulled
	link string
}

//GetPublishTime returns the publish time as a time object
func (r *RSSFeedEntry) GetPublishTime() (time.Time, error) {
	return time.Parse("Mon, 02 Jan 2006 15:04:05 MST", r.PubDateStr)
}

//Download returns a IO read Write closer of the download stream for an rss feed entry
func (r *RSSFeedEntry) Download(i *Invoker) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", r.link, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("api_key", i.APIKey)
	req.URL.RawQuery = q.Encode()

	i.requestLimiter()
	res, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}

//RSSChannel a Giant bomb RSS channel
type RSSChannel struct {
	Title   string         `xml:"title"`
	Entries []RSSFeedEntry `xml:"item"`
}

type rssBase struct {
	Channel RSSChannel `xml:"channel"`
}

//GetPodcasts returns the RSSChannel Feed
func (i *Invoker) GetPodcasts(feed string) (*RSSChannel, error) {
	var middle string
	if feed == "bombcast" {
		middle = "feeds"
		feed = "podcast"
	} else {
		middle = "podcast-xml"
	}
	url := fmt.Sprintf("%s/%s/%s/", i.Endpoint, middle, feed)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("api_key", i.APIKey)
	req.URL.RawQuery = q.Encode()

	i.requestLimiter()
	res, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	bodyXML, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var rss rssBase
	err = xml.Unmarshal([]byte(bodyXML), &rss)
	if err != nil {
		return nil, err
	}

	for i := range rss.Channel.Entries {
		guid := strings.Split(rss.Channel.Entries[i].GUID, "-")[1]
		rss.Channel.Entries[i].link = fmt.Sprintf(
			"https://dts.podtrac.com/redirect.mp3/www.giantbomb.com/podcasts/download/%s/audio.mp3",
			guid,
		)
	}

	return &rss.Channel, nil
}
