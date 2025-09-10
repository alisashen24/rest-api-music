package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/samber/lo"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var db *sql.DB

// album represents data about a record album.
type Album struct {
	ID      int64   `json:"id"`
	Title   string  `json:"title"`
	Price   float64 `json:"price"`
	LabelId int64   `json:"label_id"`
}
type Artist struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	AlbumId int64  `json:"album_id"`
}

type Label struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

type AlbumOutput struct {
	Album
	Artists []Artist `json:"artists"`
	Label   Label    `json:"label"`
}

func getLabelsByLabelIds(labelIds []int64) ([]Label, error) {
	var labels []Label
	rows, err := db.Query(
		fmt.Sprintf(
			"SELECT * FROM labels WHERE id in (%s)",
			strings.Join(lo.Map(labelIds, func(labelId int64, i int) string {
				return strconv.FormatInt(labelId, 10)
			}), ","),
		),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var label Label
		err := rows.Scan(&label.ID, &label.Name, &label.Country)
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	return labels, nil
}

func getArtistsByAlbumIds(albumIds []int64) ([]Artist, error) {
	var artists []Artist
	rows, err := db.Query(
		fmt.Sprintf(
			"SELECT * FROM artists WHERE album_id in (%s)",
			strings.Join(lo.Map(albumIds, func(albumId int64, i int) string {
				return strconv.FormatInt(albumId, 10)
			}), ","),
		),
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var artist Artist
		err := rows.Scan(&artist.ID, &artist.Name, &artist.AlbumId)
		if err != nil {
			return nil, err
		}
		artists = append(artists, artist)
	}
	return artists, nil
}

func getAlbums(c *gin.Context) {
	var albums []Album
	var albumIds []int64
	var labelIds []int64
	q := c.Query("q")
	//where title ilike '%?%'
	queryString := "SELECT * FROM albums"
	if len(q) > 0 {
		queryString += fmt.Sprintf(" WHERE title ILIKE '%%%s%%'", q)
	}
	rows, err := db.Query(queryString)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var album Album
		err := rows.Scan(&album.ID, &album.Title, &album.Price, &album.LabelId)
		if err != nil {
			c.IndentedJSON(500, gin.H{"error": err.Error()})
			return
		}
		albumIds = append(albumIds, album.ID)
		albums = append(albums, album)
		labelIds = append(labelIds, album.LabelId)
	}

	artistsList, err := getArtistsByAlbumIds(albumIds)
	if err != nil {
		c.IndentedJSON(500, gin.H{"error": err.Error()})
		return
	}

	artistsByAlbumIdMap := map[int64][]Artist{}
	for _, artist := range artistsList {
		artistsByAlbumIdMap[artist.AlbumId] = append(artistsByAlbumIdMap[artist.AlbumId], artist)
	}

	labelIdsMap := map[int64]bool{}
	for _, labelId := range labelIds {
		labelIdsMap[labelId] = true
	}
	labelIds = []int64{}
	for labelId, _ := range labelIdsMap {
		labelIds = append(labelIds, labelId)
	}

	labelsList, err := getLabelsByLabelIds(labelIds)
	if err != nil {
		c.IndentedJSON(500, gin.H{"error": err.Error()})
		return
	}

	labelsListMap := map[int64]Label{}
	for _, label := range labelsList {
		labelsListMap[label.ID] = label
	}

	albumOutputs := make([]*AlbumOutput, len(albums))
	for i, album := range albums {
		artists := artistsByAlbumIdMap[album.ID]
		label := labelsListMap[album.LabelId]
		albumOutputs[i] = &AlbumOutput{
			Album:   album,
			Artists: artists,
			Label:   label,
		}
		if err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.IndentedJSON(http.StatusOK, albumOutputs)
}

func getAlbumById(c *gin.Context) {
	id := c.Param("id")
	row := db.QueryRow("SELECT * FROM albums WHERE id=?", id)
	var album Album
	err := row.Scan(&album.ID, &album.Title, &album.Price, &album.LabelId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"message": "album not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	artists, err := getArtistsByAlbumIds([]int64{album.ID})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	labels, err := getLabelsByLabelIds([]int64{album.LabelId})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	albumOutput := AlbumOutput{
		Album:   album,
		Artists: artists,
		Label:   labels[0],
	}

	c.JSON(http.StatusOK, albumOutput)
}

func postAlbums(c *gin.Context) {
	var album Album
	if err := c.BindJSON(&album); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := db.Exec("INSERT INTO albums (title,price,label_id) VALUES(?,?,?) ", album.Title, album.Price, album.LabelId)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	row := db.QueryRow("SELECT * FROM albums WHERE id=?", id)
	err = row.Scan(&album.ID, &album.Title, &album.Price, &album.LabelId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"message": "album not found"})
		}
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	c.IndentedJSON(http.StatusCreated, album)
}

func postArtist(c *gin.Context) {
	var artist Artist

	if err := c.BindJSON(&artist); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := db.Exec("INSERT INTO artists (name, album_id) VALUES(?,?) ", artist.Name, artist.AlbumId)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	row := db.QueryRow("SELECT * FROM artists WHERE id=?", id)
	err = row.Scan(&artist.ID, &artist.Name, &artist.AlbumId)
	fmt.Println(artist)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"message": "artist not found"})
	}
	c.IndentedJSON(http.StatusCreated, artist)
}

func postLabel(c *gin.Context) {
	var label Label
	if err := c.BindJSON(&label); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "label not found"})
	}
	res, err := db.Exec("INSERT INTO labels (name, country) VALUES(?,?)", label.Name, label.Country)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	row := db.QueryRow("SELECT * FROM labels WHERE id=?", id)
	err = row.Scan(&label.ID, &label.Name, &label.Country)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "label not found"})
	}
	c.IndentedJSON(http.StatusCreated, label)
}

func putAlbums(c *gin.Context) {
	id := c.Param("id")
	var album Album
	if err := c.BindJSON(&album); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.Exec("UPDATE albums SET title=?, price=?, label_id=? WHERE id =?", album.Title, album.Price, album.LabelId, id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row := db.QueryRow("SELECT * FROM albums WHERE id=?", id)

	err = row.Scan(&album.ID, &album.Title, &album.Price, &album.LabelId)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"message": "album not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, album)
}

func deleteAlbumsById(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec("DELETE FROM albums WHERE id=?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"message": "album not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "album deleted"})
}

func main() {
	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("DBUSER")
	cfg.Passwd = os.Getenv("DBPASS")
	cfg.Net = "tcp"
	cfg.Addr = "127.0.0.1:3306"
	cfg.DBName = "recordings"

	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()
	router.GET("/albums", getAlbums)
	router.GET("/albums/:id", getAlbumById)
	router.POST("/albums", postAlbums)
	router.POST("/artists", postArtist)
	router.POST("/labels", postLabel)
	router.PUT("/albums/:id", putAlbums)
	router.DELETE("/albums/:id", deleteAlbumsById)
	router.Run("localhost:8080")
}
