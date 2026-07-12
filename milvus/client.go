package milvus

import (
	"context"
	"fmt"
	"log"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

// ChunkDoc is a text chunk with its embedding and per-chunk metadata.
type ChunkDoc struct {
	ID       string         // primary key (VARCHAR)
	Content  string         // chunk text (VARCHAR)
	Vector   []float64      // dense embedding vector
	MetaData map[string]any // book_id, chapter_number, chunk_index, etc.
}

// SearchResult holds a retrieved chunk with its distance.
type SearchResult struct {
	ID         string  `json:"id"`
	Content    string  `json:"content"`
	ChapterNum int64   `json:"chapter_number"`
	BookID     int64   `json:"book_id"`
	Distance   float64 `json:"distance"`
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client wraps the Milvus v2 client for novel chunk operations.
type Client struct {
	c        *milvusclient.Client
	collName string
	dim      int
}

// New creates a new Milvus client and ensures the collection exists.
func New(ctx context.Context, address, username, password, dbName, collName string, dim int) (*Client, error) {
	cfg := &milvusclient.ClientConfig{
		Address: address,
		DBName:  dbName,
	}

	c, err := milvusclient.New(ctx, cfg)
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("create milvus client: %w", err)
	}

	mc := &Client{
		c:        c,
		collName: collName,
		dim:      dim,
	}

	if err := mc.ensureCollection(ctx); err != nil {
		c.Close(ctx)
		return nil, fmt.Errorf("ensure collection: %w", err)
	}

	log.Printf("Milvus connected: %s, db=%s, collection=%s", address, dbName, collName)
	return mc, nil
}

// Close shuts down the Milvus client connection.
func (mc *Client) Close(ctx context.Context) error {
	if mc.c != nil {
		return mc.c.Close(ctx)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Write
// ---------------------------------------------------------------------------

// StoreChunks inserts a batch of chunk documents into the collection.
// Documents sharing the same ID are overwritten by Milvus (upsert semantics).
func (mc *Client) StoreChunks(ctx context.Context, docs []ChunkDoc) error {
	if len(docs) == 0 {
		return nil
	}

	// Filter out unusable chunks.
	valid := make([]ChunkDoc, 0, len(docs))
	for _, d := range docs {
		if len(d.Content) < 50 || len(d.Vector) == 0 {
			continue
		}
		valid = append(valid, d)
	}
	if len(valid) == 0 {
		return nil
	}

	// Build columnar data.
	ids := make([]string, len(valid))
	contents := make([]string, len(valid))
	bookIDs := make([]int64, len(valid))
	chapNums := make([]int64, len(valid))
	chunkIdxs := make([]int64, len(valid))
	vectors := make([][]float32, len(valid))

	for i, d := range valid {
		ids[i] = d.ID
		contents[i] = d.Content
		vectors[i] = float64ToFloat32(d.Vector)

		if v, ok := d.MetaData["book_id"]; ok {
			bookIDs[i] = toInt64(v)
		}
		if v, ok := d.MetaData["chapter_number"]; ok {
			chapNums[i] = toInt64(v)
		}
		if v, ok := d.MetaData["chunk_index"]; ok {
			chunkIdxs[i] = toInt64(v)
		}
	}

	_, err := mc.c.Insert(ctx, milvusclient.NewColumnBasedInsertOption(mc.collName,
		column.NewColumnVarChar("id", ids),
		column.NewColumnVarChar("content", contents),
		column.NewColumnInt64("book_id", bookIDs),
		column.NewColumnInt64("chapter_num", chapNums),
		column.NewColumnInt64("chunk_index", chunkIdxs),
		column.NewColumnFloatVector("vector", mc.dim, vectors),
	))
	if err != nil {
		return fmt.Errorf("store chunks: %w", err)
	}

	log.Printf("Stored %d chunks in Milvus", len(valid))
	return nil
}

// ---------------------------------------------------------------------------
// Read
// ---------------------------------------------------------------------------

// Search performs a vector similarity search with book_id and chapter_number
// filters. upToChapter means "return chunks from chapters <= upToChapter".
func (mc *Client) Search(ctx context.Context, queryVector []float64, bookID, upToChapter int64, limit int) ([]SearchResult, error) {
	filter := fmt.Sprintf(`book_id == %d && chapter_num <= %d`, bookID, upToChapter)

	vector := entity.FloatVector(float64ToFloat32(queryVector))

	searchOpt := milvusclient.NewSearchOption(mc.collName, limit, []entity.Vector{vector}).
		WithANNSField("vector").
		WithFilter(filter).
		WithAnnParam(index.NewHNSWAnnParam(16)).
		WithConsistencyLevel(entity.ClBounded).
		WithOutputFields("id", "content", "book_id", "chapter_num", "chunk_index")

	resultSets, err := mc.c.Search(ctx, searchOpt)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	var results []SearchResult
	for _, rs := range resultSets {
		if rs.Err != nil {
			return nil, fmt.Errorf("search result error: %w", rs.Err)
		}

		if rs.ResultCount == 0 {
			continue
		}

		for i := 0; i < rs.ResultCount; i++ {
			var r SearchResult

			if idCol := rs.GetColumn("id"); idCol != nil {
				if v, err := idCol.GetAsString(i); err == nil {
					r.ID = v
				}
			}
			if contentCol := rs.GetColumn("content"); contentCol != nil {
				if v, err := contentCol.GetAsString(i); err == nil {
					r.Content = v
				}
			}
			if bookIDCol := rs.GetColumn("book_id"); bookIDCol != nil {
				if v, err := getInt64(bookIDCol, i); err == nil {
					r.BookID = v
				}
			}
			if chapCol := rs.GetColumn("chapter_num"); chapCol != nil {
				if v, err := getInt64(chapCol, i); err == nil {
					r.ChapterNum = v
				}
			}

			if i < len(rs.Scores) {
				r.Distance = float64(rs.Scores[i])
			}

			results = append(results, r)
		}
	}

	return results, nil
}

// getInt64 extracts an int64 value from a column at the given index.
func getInt64(col column.Column, idx int) (int64, error) {
	v, err := col.Get(idx)
	if err != nil {
		return 0, err
	}
	switch val := v.(type) {
	case int64:
		return val, nil
	case int32:
		return int64(val), nil
	case int:
		return int64(val), nil
	case float64:
		return int64(val), nil
	default:
		return 0, fmt.Errorf("unexpected type %T for int64 column", v)
	}
}

// ---------------------------------------------------------------------------
// Collection management
// ---------------------------------------------------------------------------

// HasCollection checks whether the collection exists.
func (mc *Client) HasCollection(ctx context.Context) bool {
	if mc.c == nil {
		return false
	}
	has, err := mc.c.HasCollection(ctx, milvusclient.NewHasCollectionOption(mc.collName))
	return err == nil && has
}

// ensureCollection creates the collection with the expected schema and index,
// then loads it into memory. If it already exists it just loads it.
func (mc *Client) ensureCollection(ctx context.Context) error {
	has, err := mc.c.HasCollection(ctx, milvusclient.NewHasCollectionOption(mc.collName))
	if err != nil {
		return fmt.Errorf("has collection: %w", err)
	}
	if has {
		// Already exists — just make sure it's loaded.
		task, err := mc.c.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(mc.collName))
		if err != nil {
			log.Printf("warning: load collection: %v", err)
			return nil
		}
		return task.Await(ctx)
	}

	// Build schema programmatically.
	schema := entity.NewSchema().
		WithName(mc.collName).
		WithDescription("Novel text chunks with embeddings for RAG retrieval").
		WithAutoID(false).
		WithDynamicFieldEnabled(true).
		WithField(entity.NewField().
			WithName("id").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(256).
			WithIsPrimaryKey(true),
		).
		WithField(entity.NewField().
			WithName("vector").
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(mc.dim)),
		).
		WithField(entity.NewField().
			WithName("content").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(65535),
		).
		WithField(entity.NewField().
			WithName("book_id").
			WithDataType(entity.FieldTypeInt64),
		).
		WithField(entity.NewField().
			WithName("chapter_num").
			WithDataType(entity.FieldTypeInt64),
		).
		WithField(entity.NewField().
			WithName("chunk_index").
			WithDataType(entity.FieldTypeInt64),
		)

	// Create collection.
	if err := mc.c.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(mc.collName, schema).
		WithConsistencyLevel(entity.ClBounded),
	); err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	// Create HNSW index on vector field.
	hnswIdx := index.NewHNSWIndex(entity.COSINE, 16, 200)
	idxTask, err := mc.c.CreateIndex(ctx, milvusclient.NewCreateIndexOption(mc.collName, "vector", hnswIdx))
	if err != nil {
		return fmt.Errorf("create HNSW index: %w", err)
	}
	if err := idxTask.Await(ctx); err != nil {
		return fmt.Errorf("await HNSW index: %w", err)
	}

	// Create flat indexes on filter columns.
	for _, fieldName := range []string{"book_id", "chapter_num"} {
		flatIdx := index.NewFlatIndex(entity.L2)
		task, err := mc.c.CreateIndex(ctx, milvusclient.NewCreateIndexOption(mc.collName, fieldName, flatIdx))
		if err != nil {
			log.Printf("warning: create flat index for %s: %v", fieldName, err)
			continue
		}
		if err := task.Await(ctx); err != nil {
			log.Printf("warning: await flat index for %s: %v", fieldName, err)
		}
	}

	// Load collection.
	loadTask, err := mc.c.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(mc.collName))
	if err != nil {
		return fmt.Errorf("load collection: %w", err)
	}
	if err := loadTask.Await(ctx); err != nil {
		return fmt.Errorf("await load: %w", err)
	}

	log.Printf("Collection %s created, indexed, and loaded", mc.collName)
	return nil
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

// float64ToFloat32 converts []float64 to []float32 for the Milvus SDK.
func float64ToFloat32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i, f := range v {
		out[i] = float32(f)
	}
	return out
}

// toInt64 converts a map[string]any value to int64, handling float64 (from JSON)
// and other numeric types.
func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case int32:
		return int64(val)
	case int16:
		return int64(val)
	case int8:
		return int64(val)
	case uint:
		return int64(val)
	case uint64:
		return int64(val)
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// Drop collection (for testing / reset)
// ---------------------------------------------------------------------------

// DropCollection removes the entire collection.
func (mc *Client) DropCollection(ctx context.Context) error {
	if mc.c == nil {
		return nil
	}
	return mc.c.DropCollection(ctx, milvusclient.NewDropCollectionOption(mc.collName))
}
