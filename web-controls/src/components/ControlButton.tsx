import type { JSX } from 'solid-js'

type ControlButtonProps = {
  label: string
  background: string
  color: string
  icon?: JSX.Element
  disabled?: boolean
  isActive?: boolean
  style?: JSX.CSSProperties
  onClick: () => void
}

export const ControlButton = (props: ControlButtonProps) => {
  return (
    <button
      class={`control-button${props.isActive ? ' is-active' : ''}`}
      style={{
        'background-color': props.background,
        color: props.color,
        ...(props.style ?? {}),
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
