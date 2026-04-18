import type { JSX } from 'solid-js'

type ControlButtonProps = {
  label: string
  background: string
  color: string
  icon?: JSX.Element
  disabled?: boolean
  onClick: () => void
}

export const ControlButton = (props: ControlButtonProps) => {
  return (
    <button
      class="control-button"
      style={{
        'background-color': props.background,
        color: props.color,
      }}
      disabled={props.disabled}
      onClick={() => props.onClick()}
    >
      <span class="control-button-content">
        {props.icon}
        <span>{props.label}</span>
      </span>
    </button>
  )
}

