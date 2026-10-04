package picker

// PickFolder asks for a folder and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickFolder(title string) (string, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle(title).
		CanChooseDirectories(true).
		CanChooseFiles(false)
	d.AttachToWindow(s.App.Window.Current())
	return d.PromptForSingleSelection()
}
