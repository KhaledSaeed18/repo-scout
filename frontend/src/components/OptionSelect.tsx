import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui'

interface Option {
  value: string
  label: string
}

/** A compact select over a fixed list of options, labeled for screen readers. */
export default function OptionSelect({
  label,
  value,
  options,
  onChange,
  className = 'w-40',
}: {
  label: string
  value: string
  options: Option[]
  onChange: (value: string) => void
  className?: string
}) {
  return (
    <Select items={options} value={value} onValueChange={(v) => v && onChange(v)}>
      <SelectTrigger aria-label={label} className={className}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
