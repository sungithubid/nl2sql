package vectorstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/google/uuid"
	qdrant "github.com/qdrant/go-client/qdrant"
)

// Payload key 常量，与 eino-ext qdrant indexer/retriever 保持一致
const (
	PayloadKeyContent  = "content"
	PayloadKeyMetadata = "metadata"
)

// Document 向量存储的文档结构（替代 eino/schema.Document）
type Document struct {
	ID       string
	Content  string
	MetaData map[string]any
}

// VectorStore 封装 qdrant 向量数据库的存储和检索操作
type VectorStore struct {
	client    *qdrant.Client
	embedder  embedding.Embedder
	dimension int
}

// NewVectorStore 创建 VectorStore 实例
func NewVectorStore(client *qdrant.Client, embedder embedding.Embedder, dimension int) *VectorStore {
	return &VectorStore{
		client:    client,
		embedder:  embedder,
		dimension: dimension,
	}
}

// EmbedQuery 将文本转为向量（float32），用于预计算后并发查询
func (vs *VectorStore) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	vectors, err := vs.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("unexpected embedding result length: got %d, expected 1", len(vectors))
	}
	return float64ToFloat32(vectors[0]), nil
}

// Store 向量化并存储文档到指定 collection
// 如果 collection 不存在会自动创建
func (vs *VectorStore) Store(ctx context.Context, collection string, docs []*Document) ([]string, error) {
	if err := vs.ensureCollection(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to ensure collection %s: %w", collection, err)
	}

	// 为没有 ID 的文档生成 UUID
	for _, doc := range docs {
		if doc.ID == "" {
			doc.ID = uuid.New().String()
		}
	}

	// 批量 embedding
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}
	vectors, err := vs.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed documents: %w", err)
	}
	if len(vectors) != len(docs) {
		return nil, fmt.Errorf("embedding result length mismatch: got %d, expected %d", len(vectors), len(docs))
	}

	// 构建 qdrant points
	points := make([]*qdrant.PointStruct, 0, len(docs))
	for i, doc := range docs {
		payload := map[string]any{
			PayloadKeyContent: doc.Content,
		}
		if doc.MetaData != nil {
			payload[PayloadKeyMetadata] = doc.MetaData
		}

		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewID(doc.ID),
			Vectors: qdrant.NewVectors(float64ToFloat32(vectors[i])...),
			Payload: qdrant.NewValueMap(payload),
		})
	}

	// 批量写入
	_, err = vs.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         points,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert points to %s: %w", collection, err)
	}

	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}
	return ids, nil
}

// Retrieve 文本查询检索（内含 embedding 步骤）
func (vs *VectorStore) Retrieve(ctx context.Context, collection string, query string, topK int) ([]*Document, error) {
	vector, err := vs.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	return vs.RetrieveByVector(ctx, collection, vector, topK)
}

// RetrieveByVector 使用预计算的向量直接检索（跳过 embedding）
func (vs *VectorStore) RetrieveByVector(ctx context.Context, collection string, vector []float32, topK int) ([]*Document, error) {
	if topK <= 0 {
		topK = 5
	}

	resp, err := vs.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQueryDense(vector),
		Limit:          qdrant.PtrOf(uint64(topK)),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant query failed for collection %s: %w", collection, err)
	}

	docs := make([]*Document, 0, len(resp))
	for _, pt := range resp {
		doc := &Document{
			ID: pt.Id.GetUuid(),
		}
		if val, ok := pt.Payload[PayloadKeyContent]; ok {
			doc.Content = val.GetStringValue()
		}
		if val, ok := pt.Payload[PayloadKeyMetadata]; ok {
			doc.MetaData = structValueToMap(val.GetStructValue())
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// RetrieveContent 便捷方法：检索并拼接所有文档内容为字符串
func (vs *VectorStore) RetrieveContent(ctx context.Context, collection string, query string, topK int) (string, error) {
	docs, err := vs.Retrieve(ctx, collection, query, topK)
	if err != nil {
		return "", err
	}
	return joinDocContents(docs), nil
}

// RetrieveContentByVector 便捷方法：使用向量检索并拼接所有文档内容
func (vs *VectorStore) RetrieveContentByVector(ctx context.Context, collection string, vector []float32, topK int) (string, error) {
	docs, err := vs.RetrieveByVector(ctx, collection, vector, topK)
	if err != nil {
		return "", err
	}
	return joinDocContents(docs), nil
}

// Delete 按 ID 从 collection 中删除文档
func (vs *VectorStore) Delete(ctx context.Context, collection string, id string) error {
	pointID := &qdrant.PointId{
		PointIdOptions: &qdrant.PointId_Uuid{Uuid: id},
	}
	_, err := vs.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{pointID},
				},
			},
		},
	})
	return err
}

// List 列出 collection 中的所有文档（带分页限制）
func (vs *VectorStore) List(ctx context.Context, collection string, limit uint32) ([]*Document, error) {
	if limit == 0 {
		limit = 100
	}

	result, err := vs.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: collection,
		WithPayload:    qdrant.NewWithPayload(true),
		Limit:          qdrant.PtrOf(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list documents from %s: %w", collection, err)
	}

	docs := make([]*Document, 0, len(result))
	for _, pt := range result {
		doc := &Document{
			ID: pt.Id.GetUuid(),
		}
		if val, ok := pt.Payload[PayloadKeyContent]; ok {
			doc.Content = val.GetStringValue()
		}
		if val, ok := pt.Payload[PayloadKeyMetadata]; ok {
			doc.MetaData = structValueToMap(val.GetStructValue())
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// ensureCollection 确保 collection 存在，不存在时自动创建
func (vs *VectorStore) ensureCollection(ctx context.Context, collection string) error {
	exists, err := vs.client.CollectionExists(ctx, collection)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return vs.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(vs.dimension),
			Distance: qdrant.Distance_Cosine,
		}),
	})
}

// --- 工具函数 ---

func float64ToFloat32(v []float64) []float32 {
	f := make([]float32, len(v))
	for i, x := range v {
		f[i] = float32(x)
	}
	return f
}

func joinDocContents(docs []*Document) string {
	parts := make([]string, 0, len(docs))
	for _, doc := range docs {
		if doc.Content != "" {
			parts = append(parts, doc.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}

// structValueToMap 将 qdrant Struct 转为 Go map
func structValueToMap(s *qdrant.Struct) map[string]any {
	if s == nil {
		return nil
	}
	result := make(map[string]any, len(s.Fields))
	for k, v := range s.Fields {
		result[k] = qdrantValueToInterface(v)
	}
	return result
}

// qdrantValueToInterface 将 qdrant Value 转为 Go interface
func qdrantValueToInterface(v *qdrant.Value) any {
	if v == nil {
		return nil
	}
	switch v.Kind.(type) {
	case *qdrant.Value_StringValue:
		return v.GetStringValue()
	case *qdrant.Value_DoubleValue:
		return v.GetDoubleValue()
	case *qdrant.Value_IntegerValue:
		return v.GetIntegerValue()
	case *qdrant.Value_BoolValue:
		return v.GetBoolValue()
	case *qdrant.Value_StructValue:
		return structValueToMap(v.GetStructValue())
	case *qdrant.Value_ListValue:
		list := v.GetListValue().Values
		result := make([]any, len(list))
		for i, item := range list {
			result[i] = qdrantValueToInterface(item)
		}
		return result
	default:
		return nil
	}
}
