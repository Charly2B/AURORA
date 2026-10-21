import { describe, expect, it } from 'vitest'
import { audioLabel, clock, episodeCode, remaining, runtime, sameLanguage } from '../lib/format'

describe('format', () => {
  it('horloge', () => {
    expect(clock(0)).toBe('0:00')
    expect(clock(65.9)).toBe('1:05')
    expect(clock(3725)).toBe('1:02:05')
    expect(clock(NaN)).toBe('0:00')
  })
  it('durées', () => {
    expect(runtime(95)).toBe('1 h 35')
    expect(runtime(undefined, 2700)).toBe('45 min')
    expect(runtime()).toBe('')
    expect(remaining(10, 40)).toBe("Moins d'une minute restante")
    expect(remaining(0, 7200)).toBe('2 h 00 restantes')
  })
  it('langues ISO 639-1 / 639-2', () => {
    expect(sameLanguage('fre', 'fra')).toBe(true)
    expect(sameLanguage('fr', 'fre')).toBe(true)
    expect(sameLanguage('eng', 'fre')).toBe(false)
    expect(sameLanguage(undefined, 'fre')).toBe(false)
  })
  it('libellés', () => {
    expect(episodeCode(1, 3)).toBe('S01 E03')
    expect(episodeCode(2, 1, 2)).toBe('S02 E01-E02')
    expect(episodeCode(0, 5)).toBe('Spécial E05')
    expect(audioLabel({ codec: 'eac3', channels: 6, language: 'fre', title: '', atmos: true })).toBe('Français · Dolby Digital+ · 5.1 · Atmos')
  })
})
