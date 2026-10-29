// Package nxm registers Mortar as the system handler of the nxm:// scheme and forwards links to the previous handler.
package nxm

// ReasonRestricted is a mod whose author forbids downloads outside its site's own page. It sits beside go-nexusmods'
// refusal reasons because only Mortar's nxmsvc produces it.
const ReasonRestricted = "restricted"
