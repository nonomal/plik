import { describe, it, expect, beforeEach } from 'vitest'
import { config, isFeatureEnabled, isFeatureForced, isFeatureDefaultOn } from '../config.js'

// The config object is reactive and shared — reset feature flags before each test.
beforeEach(() => {
    // Reset all feature flags to a known state
    config.feature_one_shot = 'default'
    config.feature_stream = 'default'
    config.feature_password = 'default'
    config.feature_e2ee = 'enabled'
    config.feature_authentication = 'disabled'
    config.feature_comments = 'disabled'
})

// ── isFeatureEnabled ──

describe('isFeatureEnabled', () => {
    it('returns false when "disabled"', () => {
        config.feature_one_shot = 'disabled'
        expect(isFeatureEnabled('one_shot')).toBe(false)
    })

    it('returns true when "enabled"', () => {
        config.feature_one_shot = 'enabled'
        expect(isFeatureEnabled('one_shot')).toBe(true)
    })

    it('returns true when "default"', () => {
        config.feature_one_shot = 'default'
        expect(isFeatureEnabled('one_shot')).toBe(true)
    })

    it('returns true when "forced"', () => {
        config.feature_one_shot = 'forced'
        expect(isFeatureEnabled('one_shot')).toBe(true)
    })
})

// ── isFeatureForced ──

describe('isFeatureForced', () => {
    it('returns true only when "forced"', () => {
        config.feature_stream = 'forced'
        expect(isFeatureForced('stream')).toBe(true)
    })

    it('returns false for "enabled"', () => {
        config.feature_stream = 'enabled'
        expect(isFeatureForced('stream')).toBe(false)
    })

    it('returns false for "default"', () => {
        config.feature_stream = 'default'
        expect(isFeatureForced('stream')).toBe(false)
    })

    it('returns false for "disabled"', () => {
        config.feature_stream = 'disabled'
        expect(isFeatureForced('stream')).toBe(false)
    })
})

// ── isFeatureDefaultOn ──

describe('isFeatureDefaultOn', () => {
    it('returns true for "default"', () => {
        config.feature_password = 'default'
        expect(isFeatureDefaultOn('password')).toBe(true)
    })

    it('returns true for "forced"', () => {
        config.feature_password = 'forced'
        expect(isFeatureDefaultOn('password')).toBe(true)
    })

    it('returns false for "enabled"', () => {
        config.feature_password = 'enabled'
        expect(isFeatureDefaultOn('password')).toBe(false)
    })

    it('returns false for "disabled"', () => {
        config.feature_password = 'disabled'
        expect(isFeatureDefaultOn('password')).toBe(false)
    })
})
