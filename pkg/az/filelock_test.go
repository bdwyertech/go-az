package az

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("file locks", func() {
	var dir, lockPath string

	BeforeEach(func() {
		dir = useTempCredDir()
		lockPath = filepath.Join(dir, "test.lock")
	})

	It("releases the lock when fn returns an error", func() {
		boom := errors.New("boom")
		Expect(withExclusiveLock(context.Background(), lockPath, func() error {
			return boom
		})).To(MatchError(boom))

		// A second acquisition must succeed promptly, proving release.
		done := make(chan struct{})
		go func() {
			defer close(done)
			Expect(withExclusiveLock(context.Background(), lockPath, func() error {
				return nil
			})).To(Succeed())
		}()
		Eventually(done, time.Second).Should(BeClosed())
	})

	It("releases the lock when fn panics", func() {
		Expect(func() {
			_ = withExclusiveLock(context.Background(), lockPath, func() error {
				panic("kaboom")
			})
		}).To(Panic())

		done := make(chan struct{})
		go func() {
			defer close(done)
			Expect(withExclusiveLock(context.Background(), lockPath, func() error {
				return nil
			})).To(Succeed())
		}()
		Eventually(done, time.Second).Should(BeClosed())
	})

	It("never lets two exclusive holders overlap", func() {
		var inside, maxSeen int32
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					Expect(withExclusiveLock(context.Background(), lockPath, func() error {
						n := atomic.AddInt32(&inside, 1)
						for {
							m := atomic.LoadInt32(&maxSeen)
							if n <= m || atomic.CompareAndSwapInt32(&maxSeen, m, n) {
								break
							}
						}
						time.Sleep(time.Millisecond)
						atomic.AddInt32(&inside, -1)
						return nil
					})).To(Succeed())
				}
			}()
		}
		wg.Wait()
		Expect(atomic.LoadInt32(&maxSeen)).To(Equal(int32(1)))
	})

	It("permits concurrent shared readers", func() {
		release := make(chan struct{})
		entered := make(chan struct{}, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				Expect(withSharedLock(context.Background(), lockPath, func() error {
					entered <- struct{}{}
					<-release
					return nil
				})).To(Succeed())
			}()
		}
		Eventually(func() int { return len(entered) }, time.Second).Should(Equal(2))
		close(release)
		wg.Wait()
	})

	It("returns the context error instead of blocking forever", func() {
		held := make(chan struct{})
		release := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			Expect(withExclusiveLock(context.Background(), lockPath, func() error {
				close(held)
				<-release
				return nil
			})).To(Succeed())
		}()
		<-held

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		err := withExclusiveLock(ctx, lockPath, func() error {
			Fail("fn must not run when the lock cannot be acquired")
			return nil
		})
		Expect(err).To(MatchError(context.DeadlineExceeded))

		close(release)
		wg.Wait()
	})

	It("creates the lock file 0600 inside a directory created 0700", func() {
		nested := filepath.Join(dir, "nested")
		prev := setUmask(0777)
		defer setUmask(prev)

		Expect(withExclusiveLock(context.Background(), filepath.Join(nested, "a.lock"), func() error {
			return nil
		})).To(Succeed())

		di, err := os.Stat(nested)
		Expect(err).NotTo(HaveOccurred())
		Expect(di.Mode().Perm()).To(Equal(os.FileMode(0700)))

		fi, err := os.Stat(filepath.Join(nested, "a.lock"))
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode().Perm()).To(Equal(os.FileMode(0600)))
	})
})
