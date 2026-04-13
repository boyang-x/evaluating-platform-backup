package crypto

// SecureBuffer 安全缓冲区，Close 时自动清零内存
type SecureBuffer struct {
	data []byte
}

// NewSecureBuffer 创建安全缓冲区
func NewSecureBuffer(data []byte) *SecureBuffer {
	return &SecureBuffer{data: data}
}

// Bytes 返回缓冲区数据
func (b *SecureBuffer) Bytes() []byte {
	return b.data
}

// Len 返回数据长度
func (b *SecureBuffer) Len() int {
	return len(b.data)
}

// Close 将数据全部置零后释放
func (b *SecureBuffer) Close() {
	for i := range b.data {
		b.data[i] = 0
	}
	b.data = nil
}
