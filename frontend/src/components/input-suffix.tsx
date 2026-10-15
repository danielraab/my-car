// The currency or unit shown at the end of an input, such as "€" or "km".
// Its id belongs in the input's aria-describedby so the unit is announced.
export function InputSuffix({
  id,
  children,
}: {
  id: string
  children: string
}) {
  return (
    <span className="input-suffix" id={id}>
      {children}
    </span>
  )
}
