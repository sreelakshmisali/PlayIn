export type AuthStackParamList = {
  Login: undefined
  Register: undefined
}

export type PlayerTabParamList = {
  Home: undefined
  Turfs: undefined
  Profile: undefined
  Account: undefined
}

export type PlayerStackParamList = {
  PlayerTabs: undefined
  TurfDetail: { turfId: string }
  PlayerProfileEdit: undefined
}

export type OwnerTabParamList = {
  MyTurfs: undefined
  Turfs: undefined
  Profile: undefined
  Account: undefined
}

export type OwnerStackParamList = {
  OwnerTabs: undefined
  TurfDetail: { turfId: string }
  OwnerProfileEdit: undefined
  /** Absent turfId means "create a new turf". */
  OwnerTurfEdit: { turfId?: string }
}
