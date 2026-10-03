package panel

func (d *Driver) start() error {
	d.resetPower()
	if err := d.powerOn(); err != nil {
		return err
	}
	return d.configurePanel()
}

func (d *Driver) resetPower() {
	d.io.SetPower(false)
	d.io.SetCS(true)
	d.io.SetDC(false)
	d.io.SetReset(true)

	d.observe(Event{Kind: EventStep, Phase: PhaseInitialize, Step: StepPowerOff})
	d.io.Delay(powerOffDelay)
	d.io.SetPower(true)
	d.observe(Event{Kind: EventStep, Phase: PhaseInitialize, Step: StepPowerOn})
	d.io.Delay(powerSettleDelay)

	d.observe(Event{Kind: EventStep, Phase: PhaseInitialize, Step: StepReset})
	d.io.SetReset(true)
	d.io.Delay(resetHighDelay)
	d.io.SetReset(false)
	d.io.Delay(resetLowDelay)
	d.io.SetReset(true)
	d.io.Delay(resetHighDelay)

}

func (d *Driver) powerOn() error {
	if err := d.sendSmall(PhaseInitialize, StepPowerSettings, 0x01, 0x07, 0x07, 0x3f, 0x3f, 4); err != nil {
		return err
	}
	if err := d.sendSmall(PhaseInitialize, StepBoosterSoftStart, 0x06, 0x17, 0x17, 0x28, 0x17, 4); err != nil {
		return err
	}
	if err := d.sendCommand(PhasePowerOn, StepControllerPowerOn, 0x04); err != nil {
		return err
	}
	d.io.Delay(powerOnCommandDelay)
	if err := d.waitIdle(PhasePowerOn, StepControllerPowerOn, 0x04, powerOnBudget); err != nil {
		return err
	}
	return nil
}

func (d *Driver) configurePanel() error {
	if err := d.sendSmall(PhaseInitialize, StepPanelSettings, 0x00, 0x1f, 0, 0, 0, 1); err != nil {
		return err
	}
	if err := d.sendSmall(PhaseInitialize, StepResolution, 0x61, 0x03, 0x20, 0x01, 0xe0, 4); err != nil {
		return err
	}
	if err := d.sendSmall(PhaseInitialize, StepDualSPI, 0x15, 0x00, 0, 0, 0, 1); err != nil {
		return err
	}
	if err := d.sendSmall(PhaseInitialize, StepVCOM, 0x50, 0x10, 0x07, 0, 0, 2); err != nil {
		return err
	}
	if err := d.sendSmall(PhaseInitialize, StepTCON, 0x60, 0x22, 0, 0, 0, 1); err != nil {
		return err
	}

	return nil
}

func (d *Driver) finish() error {
	if err := d.sendCommand(PhaseRefresh, StepDisplayRefresh, 0x12); err != nil {
		return err
	}
	d.io.Delay(refreshCommandDelay)
	if err := d.waitIdle(PhaseRefresh, StepDisplayRefresh, 0x12, refreshBudget); err != nil {
		return err
	}

	if err := d.sendSmall(PhasePowerOff, StepVCOMOff, 0x50, 0xf7, 0, 0, 0, 1); err != nil {
		return err
	}
	if err := d.sendCommand(PhasePowerOff, StepControllerPowerOff, 0x02); err != nil {
		return err
	}
	if err := d.waitIdle(PhasePowerOff, StepControllerPowerOff, 0x02, powerOffBudget); err != nil {
		return err
	}
	if err := d.sendSmall(PhasePowerOff, StepDeepSleep, 0x07, 0xa5, 0, 0, 0, 1); err != nil {
		return err
	}

	d.observe(Event{Kind: EventComplete, Phase: PhasePowerOff, Step: StepDeepSleep})
	return nil
}
