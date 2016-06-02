type HookHandler = (arg: any) => void
type HookMap = {[key: string]: HookHandler[]}

interface ChangeEmitter {
	onChange(key: string, func: HookHandler): void
}
