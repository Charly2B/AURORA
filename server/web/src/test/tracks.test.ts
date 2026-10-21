import { describe, expect, it } from 'vitest'
import { initialChoice } from '../pages/Player'
import type { FileInfo, Prefs } from '../lib/types'

const file = {
  audio: [
    { index: 0, stream: 1, codec: 'eac3', profile: '', channels: 6, layout: '', language: 'eng', title: '', default: true, atmos: false },
    { index: 1, stream: 2, codec: 'ac3', profile: '', channels: 6, layout: '', language: 'fre', title: '', default: false, atmos: false },
  ],
  subtitles: [
    { index: 0, codec: 'subrip', language: 'fre', title: '', default: false, forced: true, text: true },
    { index: 1, codec: 'subrip', language: 'fre', title: '', default: false, forced: false, text: true },
    { index: 2, codec: 'hdmv_pgs_subtitle', language: 'fre', title: '', default: false, forced: false, text: false },
  ],
} as unknown as FileInfo
const base: Prefs = { home: [], autoplayNext: true, audioLang: 'fre', subLang: '', subMode: 'auto', subSize: 'm' }

describe('choix initial des pistes', () => {
  it('audio dans la langue préférée, sous-titres forcés seulement', () => {
    expect(initialChoice(file, base)).toEqual({ audioIndex: 1, subtitleIndex: 0 })
  })
  it('VO + sous-titres complets quand on préfère la VO', () => {
    expect(initialChoice(file, { ...base, audioLang: 'eng', subLang: 'fre' })).toEqual({ audioIndex: 0, subtitleIndex: 1 })
  })
  it('jamais de sous-titres en mode « off »', () => {
    expect(initialChoice(file, { ...base, subMode: 'off' }).subtitleIndex).toBe(-1)
  })
  it("ne choisit jamais d'office un sous-titre image (incrustation coûteuse)", () => {
    const onlyImage = { ...file, subtitles: [file.subtitles[2]] } as FileInfo
    expect(initialChoice(onlyImage, { ...base, subMode: 'always' }).subtitleIndex).toBe(-1)
  })
})
