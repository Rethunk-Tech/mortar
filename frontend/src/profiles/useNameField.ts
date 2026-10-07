import { useEffect, useState } from 'react'

// The Edit profile form's name and the message the store gave when it refused it; both reset each time the form opens.
export function useNameField(current: string, open: boolean) {
  const [name, setValue] = useState(current)
  const [nameError, setNameError] = useState('')
  useEffect(() => {
    if (open) {
      setValue(current)
      setNameError('')
    }
  }, [open, current])
  // Typing clears the refusal it answers.
  const setName = (value: string) => {
    setNameError('')
    setValue(value)
  }
  return { name, setName, nameError, setNameError }
}
