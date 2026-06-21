export interface GuardPageDefinition {
  template: string
  strings: Record<string, Record<string, string>>
}

export type GuardPageMap = Record<string, GuardPageDefinition>
