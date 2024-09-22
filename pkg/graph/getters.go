package graph

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

)

type getOpts struct {
	source        types.Source
	author        *uuid.UUID
	chat          *string
	count         *int64
	before        *time.Time
	after         *time.Time
	sort          *types.Sort
	onlyVisible   *bool
	onlyWithMedia *bool
}

func getPosts(parentCtx context.Context, opts getOpts) ([]*model.Post, error) {
	count := 20
	if opts.count != nil && *opts.count >= 1 && *opts.count <= 20 {
		count = int(*opts.count)
	}

	onlyVisible := opts.onlyVisible != nil && *opts.onlyVisible
	onlyWithMedia := opts.onlyWithMedia != nil && *opts.onlyWithMedia

	conn := db.Get()

	wheres := []string{
		"post_source = " + db.GetPlaceholder(),
	}
	params := []any{
		opts.source,
	}

	if opts.author != nil && IsValidUUID(*opts.author) {
		params = append(params, *opts.author)
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, "post_author = ?")
		case "pgx":
			wheres = append(wheres, fmt.Sprintf("post_author = $%d", len(params)))
		}
	}

	sort := types.SortDesc
	reverseSort := false
	if opts.sort != nil {
		sort = *opts.sort
	}

	if opts.after != nil {
		params = append(params, opts.after.Unix())
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, "post_date > ?")
		case "pgx":
			wheres = append(wheres, fmt.Sprintf("post_date > $%d", len(params)))
		}
		sort = types.ReverseSort(sort)
		reverseSort = true
	}
	if opts.before != nil {
		params = append(params, opts.before.Unix())
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, "post_date < ?")
		case "pgx":
			wheres = append(wheres, fmt.Sprintf("post_date < $%d", len(params)))
		}
	}

	sortQ := "DESC"
	if sort == types.SortAsc {
		sortQ = "ASC"
	}

	if onlyWithMedia {
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, `
			(
				post_media IS NOT NULL
				AND json_array_length(post_media) > 0
				AND EXISTS (
					SELECT 1
					FROM `+models.MediaBase.TableName()+`, json_each(post_media) AS j
					WHERE media_visible = 1 AND media_id = j.value
				)
			)`)
		case "pgx":
			wheres = append(wheres, `
			(
				post_media IS NOT NULL
				AND array_length(post_media, 1) > 0
				AND array_length(post_media, 1) IS NOT NULL
				AND EXISTS (
					SELECT 1 FROM `+models.MediaBase.TableName()+` WHERE media_id = ANY(post_media) AND media_visible = 1
				)
			)`)
		}
	} else if onlyVisible {
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, `
			(
				post_media IS NULL
				OR json_array_length(post_media) = 0
				OR EXISTS (
					SELECT 1
					FROM `+models.MediaBase.TableName()+`, json_each(post_media) AS j
					WHERE media_visible = 1 AND media_id = j.value
				)
			)`)
		case "pgx":
			wheres = append(wheres, `
			(
				post_media IS NULL
				OR array_length(post_media, 1) = 0
				OR array_length(post_media, 1) IS NULL
				OR EXISTS (
					SELECT 1 FROM `+models.MediaBase.TableName()+` WHERE media_id = ANY(post_media) AND media_visible = 1
				)
			)`)
		}
	}

	var whereQ string
	if len(wheres) > 0 {
		whereQ = "WHERE " + strings.Join(wheres, " AND ")
	}

	q := `SELECT ` + postCols + `
	FROM ` + models.PostBase.TableName() + `
	` + whereQ + `
	ORDER BY post_date ` + sortQ + `
	LIMIT ` + strconv.Itoa(count)

	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()

	rows, err := conn.QueryContext(ctx, q, params...)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	posts := make([]*model.Post, count)
	i := 0
	for rows.Next() {
		p, err := parsePostRow(rows)
		if err != nil {
			return nil, fmt.Errorf("error reading row: %w", err)
		}

		if reverseSort {
			posts[count-i-1] = p
		} else {
			posts[i] = p
		}
		i++
	}
	if reverseSort {
		posts = posts[(count - i):]
	} else {
		posts = posts[:i]
	}

	return posts, nil
}

func getMessages(parentCtx context.Context, opts getOpts) ([]*model.Message, error) {
	if opts.chat == nil || *opts.chat == "" {
		return nil, errors.New("chat ID must be specified")
	}

	count := 20
	if opts.count != nil && *opts.count >= 1 && *opts.count <= 20 {
		count = int(*opts.count)
	}

	onlyWithMedia := opts.onlyWithMedia != nil && *opts.onlyWithMedia
	onlyVisible := opts.onlyVisible != nil && *opts.onlyVisible

	conn := db.Get()

	placeholders := db.GetPlaceholders(2)
	wheres := []string{
		"message_source = " + placeholders[0],
		"message_source_chat_id = " + placeholders[1],
	}
	params := []any{
		opts.source,
		*opts.chat,
	}

	sort := types.SortDesc
	reverseSort := false
	if opts.sort != nil {
		sort = *opts.sort
	}

	if opts.after != nil {
		params = append(params, opts.after.Unix())
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, "message_date > ?")
		case "pgx":
			wheres = append(wheres, fmt.Sprintf("message_date > $%d", len(params)))
		}
		sort = types.ReverseSort(sort)
		reverseSort = true
	}
	if opts.before != nil {
		params = append(params, opts.before.Unix())
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, "message_date < ?")
		case "pgx":
			wheres = append(wheres, fmt.Sprintf("message_date < $%d", len(params)))
		}
	}

	sortQ := "DESC"
	if sort == types.SortAsc {
		sortQ = "ASC"
	}

	if onlyWithMedia {
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, `
			(
				message_media IS NOT NULL
				AND json_array_length(message_media) > 0
				AND EXISTS (
					SELECT 1
					FROM `+models.MediaBase.TableName()+`, json_each(message_media) AS j
					WHERE media_visible = 1 AND media_id = j.value
				)
			)`)
		case "pgx":
			wheres = append(wheres, `
			(
				message_media IS NOT NULL
				AND array_length(message_media, 1) IS NOT NULL
				AND array_length(message_media, 1) > 0
				AND EXISTS (
					SELECT 1 FROM `+models.MediaBase.TableName()+` WHERE media_id = ANY(message_media) AND media_visible = 1
				)
			)`)
		}
	} else if onlyVisible {
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, `
			(
				message_media IS NULL
				OR json_array_length(message_media) = 0
				OR EXISTS (
					SELECT 1
					FROM `+models.MediaBase.TableName()+`, json_each(message_media) AS j
					WHERE media_visible = 1 AND media_id = j.value
				)
			)`)
		case "pgx":
			wheres = append(wheres, `
			(
				message_media IS NULL
				OR array_length(message_media, 1) IS NULL
				OR array_length(message_media, 1) = 0
				OR EXISTS (
					SELECT 1 FROM `+models.MediaBase.TableName()+` WHERE media_id = ANY(message_media) AND media_visible = 1
				)
			)`)
		}
	}

	var whereQ string
	if len(wheres) > 0 {
		whereQ = "WHERE " + strings.Join(wheres, " AND ")
	}

	q := `SELECT
		message_id, message_source, message_source_id, message_author, message_date, message_text, message_media
	FROM ` + models.MessageBase.TableName() + `
	` + whereQ + `
	ORDER BY message_date ` + sortQ + `
	LIMIT ` + strconv.Itoa(count)

	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()

	rows, err := conn.QueryContext(ctx, q, params...)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	messages := make([]*model.Message, count)
	i := 0
	for rows.Next() {
		m, err := parseMessageRow(rows)
		if err != nil {
			return nil, fmt.Errorf("error reading row: %w", err)
		}

		if reverseSort {
			messages[count-i-1] = m
		} else {
			messages[i] = m
		}
		i++
	}
	if reverseSort {
		messages = messages[(count - i):]
	} else {
		messages = messages[:i]
	}

	return messages, nil
}

func getStories(parentCtx context.Context, opts getOpts) ([]*model.tag, error) {
	count := 20
	if opts.count != nil && *opts.count >= 1 && *opts.count <= 20 {
		count = int(*opts.count)
	}

	if opts.before != nil && opts.after != nil {
		return nil, errors.New("cannot have both before and after")
	}

	conn := db.Get()

	wheres := []string{
		"tag_source = " + db.GetPlaceholder(),
	}
	params := []any{
		opts.source,
	}

	if opts.author != nil && IsValidUUID(*opts.author) {
		params = append(params, *opts.author)
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, "tag_author = ?")
		case "pgx":
			wheres = append(wheres, fmt.Sprintf("tag_author = $%d", len(params)))
		}
	}

	sort := types.SortDesc
	reverseSort := false
	if opts.sort != nil {
		sort = *opts.sort
	}

	if opts.after != nil {
		params = append(params, opts.after.Unix())
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, "tag_date > ?")
		case "pgx":
			wheres = append(wheres, fmt.Sprintf("tag_date > $%d", len(params)))
		}
		sort = types.ReverseSort(sort)
		reverseSort = true
	}
	if opts.before != nil {
		params = append(params, opts.before.Unix())
		switch conn.DriverName() {
		case "sqlite":
			wheres = append(wheres, "tag_date < ?")
		case "pgx":
			wheres = append(wheres, fmt.Sprintf("tag_date < $%d", len(params)))
		}
	}

	sortQ := "DESC"
	if sort == types.SortAsc {
		sortQ = "ASC"
	}

	var whereQ string
	if len(wheres) > 0 {
		whereQ = "WHERE " + strings.Join(wheres, " AND ")
	}

	q := `SELECT
		tag_id, tag_source, tag_source_id, tag_author, tag_date, tag_media
	FROM ` + models.tagBase.TableName() + `
	` + whereQ + `
	ORDER BY tag_date ` + sortQ + `
	LIMIT ` + strconv.Itoa(count)

	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()

	rows, err := conn.QueryContext(ctx, q, params...)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	stories := make([]*model.tag, count)
	i := 0
	for rows.Next() {
		p, err := parsetagRow(rows)
		if err != nil {
			return nil, fmt.Errorf("error reading row: %w", err)
		}
		if reverseSort {
			stories[count-i-1] = p
		} else {
			stories[i] = p
		}
		i++
	}
	if reverseSort {
		stories = stories[(count - i):]
	} else {
		stories = stories[:i]
	}

	return stories, nil
}
