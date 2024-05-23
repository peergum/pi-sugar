/*
   pi_sugar,
   Copyright (C) 2024  Phil Hilger

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package pi_sugar

import "github.com/peergum/go-rpio/v4"

//import rpio "github.com/peergum/go-rpio/v5"

type PiSugar struct {
	voltage  int
	charge   int
	power    bool
	charging bool
	model    int
}

var (
	piSugar PiSugar
)

func NewPiSugar() (*PiSugar, error) {
	if err := rpio.I2cBegin(rpio.I2c1); err != nil {
		return nil, err
	}
	return &piSugar, nil
}

func (piSugar *PiSugar) Voltage() int {
	return piSugar.voltage
}

func (piSugar *PiSugar) Charge() int {
	return piSugar.charge
}

func (piSugar *PiSugar) Charging() bool {
	return piSugar.charging
}

func (piSugar *PiSugar) Power() bool {
	return piSugar.power
}

func (piSugar *PiSugar) Refresh() {

}
