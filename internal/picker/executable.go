package picker

// PickExecutable asks for one program and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickExecutable(title string) (string, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle(title).
		CanChooseDirectories(false).
		CanChooseFiles(true)
	d.AttachToWindow(s.App.Window.Current())
	return d.PromptForSingleSelection()
}
