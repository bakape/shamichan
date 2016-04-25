interface System {
  import(name: string): Promise<any>
}

declare var System: System
