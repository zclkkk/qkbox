package qkboxd

import (
	qkboxcrypto "github.com/zclkkk/qkbox/internal/crypto"
	"github.com/zclkkk/qkbox/internal/persistence"
)

type ContentCodec struct {
	db  *persistence.DB
	key []byte
}

func (c *ContentCodec) encryptedContent(sourceType, sourceID, content string, createdAt int64) (*persistence.EncryptedContent, error) {
	iv, ct, err := qkboxcrypto.Encrypt(c.key, []byte(content))
	if err != nil {
		return nil, err
	}
	return &persistence.EncryptedContent{
		ID:         persistence.NewContentID(),
		SourceType: sourceType,
		SourceID:   sourceID,
		IV:         iv,
		Ciphertext: ct,
		CreatedAt:  createdAt,
	}, nil
}

func (c *ContentCodec) decryptContent(contentID string) (string, error) {
	content, err := c.db.GetContent(contentID)
	if err != nil {
		return "", err
	}
	if content == nil {
		return "", nil
	}
	plaintext, err := qkboxcrypto.Decrypt(c.key, content.IV, content.Ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
