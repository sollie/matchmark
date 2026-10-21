package service

func (b *Broadcaster) RetainedVersions() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.retained)
}
