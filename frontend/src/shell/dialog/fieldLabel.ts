// A select's combobox is not the element a Field's htmlFor names, so it takes its name from the label by this id.
export const labelIdOf = (htmlFor: string) => `${htmlFor}-label`
