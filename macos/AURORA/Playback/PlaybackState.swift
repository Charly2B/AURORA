enum PlaybackState: Equatable {
    case idle
    case preparing
    case playing
    case paused
    case buffering
    case ended
    case failed
}
