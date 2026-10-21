import { afterEach, describe, expect, it } from 'vitest'
import { detectCaps } from '../lib/caps'

function fakeVideo(supported: string[]): HTMLVideoElement {
  return { canPlayType: (t: string) => (supported.some((s) => t.startsWith(s)) ? 'probably' : '') } as unknown as HTMLVideoElement
}

const setVendor = (v: string) => Object.defineProperty(navigator, 'vendor', { value: v, configurable: true })

describe('détection des capacités', () => {
  afterEach(() => setVendor('Apple Computer, Inc.'))
  it('Chrome sans HEVC : H.264/VP9/AAC via MSE, pas de HEVC ni AC3', () => {
    const v = fakeVideo(['video/mp4; codecs="avc1', 'audio/mp4; codecs="mp4a.40.2"', 'video/webm', 'application/vnd.apple.mpegurl'])
    const mse = { isTypeSupported: (t: string) => t.includes('avc1') || t.includes('mp4a.40.2') || t.includes('vp9') }
    setVendor('Google Inc.')
    const c = detectCaps(v, mse)
    expect(c.hls).toBe('mse') // HLS natif ignoré hors navigateur Apple (Chromium répond « maybe » à tort)
    expect(c.video).toEqual(['h264', 'vp9']) // VP9 annoncé par le faux navigateur (WebM) et par MSE
    expect(c.audio).toEqual(['aac'])
    expect(c.containers).toContain('webm')
  })
  it('sans MSE mais HLS natif : mode natif', () => {
    const v = fakeVideo(['video/mp4; codecs="avc1', 'audio/mp4; codecs="mp4a.40.2"', 'application/vnd.apple.mpegurl'])
    const c = detectCaps(v, null)
    expect(c.hls).toBe('native')
    expect(c.video).toContain('h264')
  })
  it('Safari : HLS natif', () => {
    setVendor('Apple Computer, Inc.')
    const v = fakeVideo(['video/mp4; codecs="avc1', 'video/mp4; codecs="hvc1', 'audio/mp4; codecs="mp4a.40.2"', 'audio/mp4; codecs="ac-3"', 'application/vnd.apple.mpegurl'])
    const c = detectCaps(v, { isTypeSupported: () => false })
    expect(c.hls).toBe('native')
    expect(c.video).toEqual(['h264', 'hevc'])
    expect(c.audio).toEqual(['aac', 'ac3'])
  })
  it("un codec lisible en direct mais refusé par MSE n'est pas retenu", () => {
    setVendor('Google Inc.')
    const v = fakeVideo(['video/mp4; codecs="avc1', 'video/mp4; codecs="hvc1', 'audio/mp4; codecs="mp4a.40.2"'])
    const mse = { isTypeSupported: (t: string) => t.includes('avc1') || t.includes('mp4a') }
    expect(detectCaps(v, mse).video).toEqual(['h264'])
  })
})
